# Automated API test runner and result saver

import requests
import json
from datetime import datetime
import argparse
import os



def build_curl_command(url, method, headers, data):
	"""
	Builds a formatted curl command with proper line continuations and indentation.
	"""
	curl_lines = [f'curl -s -X {method} "{url}"']
	if headers:
		for k, v in headers.items():
			curl_lines.append(f'    -H "{k}: {v}"')
	if data is not None:
		if isinstance(data, dict):
			# Format JSON with proper indentation (4 spaces base, 2 spaces for structure)
			pretty_data = json.dumps(data, indent=2)
			# Indent each line of the JSON by 4 spaces
			indented_lines = []
			for line in pretty_data.split('\n'):
				indented_lines.append('      ' + line)
			indented_data = '\n'.join(indented_lines)
			curl_lines.append(f"    -d '\n{indented_data}\n    '")
		else:
			curl_lines.append(f"    -d '{data}'")
	# Add trailing backslash for all but last line
	for i in range(len(curl_lines)-1):
		curl_lines[i] += ' \\'
	curl_cmd = "\n".join(curl_lines) + " | jq ."
	return curl_cmd

def run_test(base_url, prefix, test):
	"""
	Executes a single API request using the given base URL, prefix, and test definition.
	Returns the curl command, response status, response body (as string), and response object.
	"""
	method = test.get("cmd", "GET").upper()
	endpoint = test.get("endpoint", "/")
	url = f"{base_url}{prefix}{endpoint}"
	headers = test.get("headers", {})
	data = test.get("data")
	req_kwargs = {}
	if headers:
		req_kwargs["headers"] = headers
	if data is not None:
		if isinstance(data, dict):
			req_kwargs["json"] = data
		else:
			req_kwargs["data"] = data
	
	curl_cmd = build_curl_command(url, method, headers, data)
	
	response_obj = None
	try:
		resp = requests.request(method, url, **req_kwargs)
		status = f"{resp.status_code} {resp.reason}"
		try:
			body = resp.json()
			response_obj = body
			body_str = json.dumps(body, indent=2)
		except Exception:
			body_str = resp.text
	except Exception as e:
		status = "ERROR"
		body_str = str(e)
	return curl_cmd, status, body_str, response_obj


def parse_args():
	"""
	Parses command-line arguments for output directory.
	"""
	parser = argparse.ArgumentParser(description="Automated API test runner")
	parser.add_argument('-o', '--outdir', default='./out', help='Output directory for result files (default: ./out)')
	parser.add_argument('-i', '--input', default=None, help='Test JSON file to run (default: all in tests dir)')
	parser.add_argument('-t', '--testdir', default='./tests', help='Directory containing test JSON files (default: ./tests)')
	return parser.parse_args()

def setup_outdir(outdir):
	"""
	Ensures the output directory exists (creates if missing).
	"""
	os.makedirs(outdir, exist_ok=True)

def format_markdown_result(curl_cmd, status, body, test_name=None, details=None, notes=None):
	"""
	Formats a single API test result as a Markdown snippet for output, with heading if test_name is given.
	Optionally includes details (before curl command) and notes (after response).
	"""
	heading = f"### {test_name}\n\n" if test_name else ""
	details_section = f"{details}\n\n" if details else ""
	notes_section = f"{notes}\n\n" if notes else ""
	return [
		heading + details_section + notes_section + f"```bash\n{curl_cmd}\n```",
		f"\n**Response ({status}):**\n",
		f"```json\n{body}\n```\n"
	]

def extract_collection_name(endpoint):
	"""
	Extracts the collection name from an endpoint.
	E.g., "/products:get" -> "products"
	"""
	if '/' in endpoint:
		endpoint = endpoint.split('?')[0]  # Remove query params
		parts = endpoint.split('/')
		for part in parts:
			if ':' in part:
				return part.split(':')[0]
	return None

def fetch_record_id(base_url, prefix, collection_name, headers):
	"""
	Fetches the first record ID from a collection by calling /{collection}:list
	"""
	try:
		list_endpoint = f"/{collection_name}:list"
		url = f"{base_url}{prefix}{list_endpoint}"
		resp = requests.get(url, headers=headers, timeout=10)
		if resp.status_code == 200:
			data = resp.json()
			# Try to find records in common response structures
			for array_key in ["data", "records", "items", "apikeys", "users"]:
				records = data.get(array_key, [])
				if records and len(records) > 0:
					# Try common ID field names
					first_record = records[0]
					return first_record.get("id", first_record.get("_id", first_record.get("ulid")))
	except Exception as e:
		pass
	return None

def extract_record_id_from_response(response_obj):
	"""
	Extracts record ID from a create response.
	Checks common patterns like data.id, record.id, id, etc.
	Also checks for arrays in apikeys, users, data, records, items fields.
	"""
	if not response_obj or not isinstance(response_obj, dict):
		return None
	
	# Try direct id field
	if "id" in response_obj:
		return response_obj["id"]
	
	# Try data.id
	if "data" in response_obj and isinstance(response_obj["data"], dict):
		if "id" in response_obj["data"]:
			return response_obj["data"]["id"]
	
	# Try record.id
	if "record" in response_obj and isinstance(response_obj["record"], dict):
		if "id" in response_obj["record"]:
			return response_obj["record"]["id"]
	
	# Try arrays in apikeys, users, data, records, items
	for array_key in ["apikeys", "users", "data", "records", "items"]:
		if array_key in response_obj and isinstance(response_obj[array_key], list) and len(response_obj[array_key]) > 0:
			# For users array, select second record if available, otherwise first
			if array_key == "users" and len(response_obj[array_key]) > 1:
				selected_item = response_obj[array_key][1]
			else:
				selected_item = response_obj[array_key][0]
			
			if isinstance(selected_item, dict) and "id" in selected_item:
				return selected_item["id"]
	
	# Try other common patterns
	for key in ["_id", "ulid", "uuid"]:
		if key in response_obj:
			return response_obj[key]
		if "data" in response_obj and isinstance(response_obj["data"], dict) and key in response_obj["data"]:
			return response_obj["data"][key]
	
	return None

def replace_record_in_test(test, record_id):
	"""
	Replaces $ULID and $NEXT_CURSOR placeholders in a single test with the actual record ID.
	Returns the placeholder type that was used ('$ULID', '$NEXT_CURSOR', or None).
	"""
	if not record_id:
		return None
	
	placeholder_used = None
	
	# Replace in endpoint
	if "endpoint" in test:
		if "$NEXT_CURSOR" in test["endpoint"]:
			test["endpoint"] = test["endpoint"].replace("$NEXT_CURSOR", record_id)
			placeholder_used = "$NEXT_CURSOR"
		elif "$ULID" in test["endpoint"]:
			test["endpoint"] = test["endpoint"].replace("$ULID", record_id)
			placeholder_used = "$ULID"
	
	# Replace in data (recursive)
	if "data" in test and test["data"]:
		data_str = json.dumps(test["data"])
		if "$NEXT_CURSOR" in data_str:
			data_str = data_str.replace("$NEXT_CURSOR", record_id)
			placeholder_used = "$NEXT_CURSOR"
		elif "$ULID" in data_str:
			data_str = data_str.replace("$ULID", record_id)
			placeholder_used = "$ULID"
		test["data"] = json.loads(data_str)
	
	return placeholder_used

def run_all_tests(tests, outdir, access_token=None, outfilename=None):
	"""
	Runs all API tests and writes Markdown output to the output file.
	Returns status (success/failure) and output file path.
	"""
	results_md = []
	docURL = tests["docURL"]
	serverURL = tests["serverURL"]
	prefix = tests.get("prefix", "")
	all_ok = True
	captured_record_id = None
	refresh_token = None
	all_access_tokens = []  # Track all access tokens used
	all_refresh_tokens = []  # Track all refresh tokens used
	if access_token:
		all_access_tokens.append(access_token)
	
	for test in tests["tests"]:
		# Replace $ULID or $NEXT_CURSOR in current test if we have a captured ID
		placeholder_type = None
		if captured_record_id:
			placeholder_type = replace_record_in_test(test, captured_record_id)
		
		# Replace $ACCESS_TOKEN in current test headers
		if access_token and 'headers' in test and 'Authorization' in test['headers']:
			test['headers']['Authorization'] = test['headers']['Authorization'].replace('$ACCESS_TOKEN', access_token)
		
		# Replace $REFRESH_TOKEN in current test data
		if refresh_token and 'data' in test and test['data']:
			data_str = json.dumps(test['data'])
			if '$REFRESH_TOKEN' in data_str:
				data_str = data_str.replace('$REFRESH_TOKEN', refresh_token)
				test['data'] = json.loads(data_str)
		
		curl_cmd, status, body, response_obj = run_test(serverURL, prefix, test)
		
		# Check if this test is a login or refresh and capture tokens
		endpoint = test.get("endpoint", "")
		method = test.get("cmd", "GET").upper()
		if response_obj and status.startswith("2") and method == "POST":
			if "auth:login" in endpoint or "auth:refresh" in endpoint:
				new_access_token = response_obj.get("access_token")
				if new_access_token:
					access_token = new_access_token
					if new_access_token not in all_access_tokens:
						all_access_tokens.append(new_access_token)
				new_refresh_token = response_obj.get("refresh_token")
				if new_refresh_token:
					refresh_token = new_refresh_token
					if new_refresh_token not in all_refresh_tokens:
						all_refresh_tokens.append(new_refresh_token)
		
		# Check if this test created/listed a record and capture the ID
		if response_obj and status.startswith("2") and not captured_record_id:
			# Try to capture ID from :create or :list endpoints
			if ":create" in endpoint or ":list" in endpoint:
				record_id = extract_record_id_from_response(response_obj)
				if record_id:
					captured_record_id = record_id
		
		# Replace actual server URL with doc URL for display
		curl_cmd_doc = curl_cmd.replace(serverURL, docURL)
		# Replace all access tokens with placeholder for documentation
		for token in all_access_tokens:
			if token:
				curl_cmd_doc = curl_cmd_doc.replace(token, "$ACCESS_TOKEN")
		# Replace all refresh tokens with placeholder for documentation
		for token in all_refresh_tokens:
			if token:
				curl_cmd_doc = curl_cmd_doc.replace(token, "$REFRESH_TOKEN")
		# Replace actual record ID with appropriate placeholder for documentation
		if captured_record_id and placeholder_type:
			curl_cmd_doc = curl_cmd_doc.replace(captured_record_id, placeholder_type)
		
		test_name = test.get("name", "").strip()
		test_details = test.get("details", None)
		test_notes = test.get("notes", None)
		
		# Only add to markdown output if test has a name
		if test_name:
			results_md.extend(format_markdown_result(curl_cmd_doc, status, body, test_name, test_details, test_notes))
		
		if not status.startswith("2"):
			all_ok = False
	markdown = "\n".join(results_md)
	if outfilename:
		with open(outfilename, "w", encoding="utf-8") as f:
			f.write(markdown)
	return ("success" if all_ok else "failure", outfilename, markdown)

def main():
	"""
	Entry point: parses arguments, ensures output directory, and runs all tests. Prints only status and markdown output.
	"""
	args = parse_args()
	setup_outdir(args.outdir)
	test_files = []
	if args.input:
		test_files = [args.input]
	else:
		# Find all .json files in testdir
		test_files = [os.path.join(args.testdir, f) for f in os.listdir(args.testdir) if f.endswith('.json')]
	for test_file in test_files:
		with open(test_file, 'r', encoding='utf-8') as f:
			tests = json.load(f)

		# Perform health check if specified
		health_endpoint = tests.get("health", "/health")
		health_url = f"{tests['serverURL']}{tests.get('prefix', '')}{health_endpoint}"
		try:
			health_resp = requests.get(health_url, timeout=5)
			if health_resp.status_code != 200:
				print(f"Skipping {test_file} [server unhealthy: {health_resp.status_code}]")
				continue
		except Exception as e:
			print(f"Skipping {test_file} [server unreachable: {e}]")
			continue

		# Check if any test uses Authorization header with $ACCESS_TOKEN
		need_token = any(
			'headers' in t and 'Authorization' in t['headers'] and '$ACCESS_TOKEN' in t['headers']['Authorization']
			for t in tests.get('tests', [])
		)
		access_token = None
		if need_token:
			# Perform login
			login_url = f"{tests['serverURL']}/auth:login"
			login_data = {
				"username": tests.get("username", "admin"),
				"password": tests.get("password", "moonadmin12#")
			}
			try:
				resp = requests.post(login_url, json=login_data, headers={"Content-Type": "application/json"})
				resp.raise_for_status()
				token_json = resp.json()
				access_token = token_json.get("access_token")
			except Exception as e:
				print(f"Login failed: {e}")
				access_token = None
		# Replace $ACCESS_TOKEN in test headers
		for t in tests.get('tests', []):
			if 'headers' in t and 'Authorization' in t['headers'] and '$ACCESS_TOKEN' in t['headers']['Authorization']:
				if access_token:
					t['headers']['Authorization'] = t['headers']['Authorization'].replace('$ACCESS_TOKEN', access_token)
				else:
					t['headers']['Authorization'] = t['headers']['Authorization'].replace('$ACCESS_TOKEN', '')

		# Output file: out/<basename>.md
		base = os.path.splitext(os.path.basename(test_file))[0]
		outfilename = os.path.join(args.outdir, f"{base}.md")
		status, outfile, markdown = run_all_tests(tests, args.outdir, access_token, outfilename)
		print("\n==============================================")
		print(f"Executed {test_file} [{status}]")
		print("==============================================\n")
		print(markdown)

if __name__ == "__main__":
	main()

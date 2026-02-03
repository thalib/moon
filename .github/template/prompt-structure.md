Refactor below ai agent prompt file

- `.github\prompts\UpdateSPEC.prompt.md`
- `.github\prompts\UpdateApiDocumentation.prompt.md`

in below structure

```markdown
## Role

[Persona/expertise level - ONLY include if it significantly changes behavior. E.g., "You are a senior security auditor" vs generic coder]

## Context

[1-3 sentences: Domain/situation background. What does the AI need to know about the environment, use case, or problem space? Keep it brief - only non-obvious information.]

## Objective

[1 sentence: Clear, measurable goal. What does success look like?]

## Instructions

[Step-by-step process OR prioritized requirements list. Use imperative verbs: "Extract", "Generate", "Validate". Order by importance or execution sequence.]

## Constraints

[Hard boundaries and requirements. Split into MUST and MUST NOT for clarity.]

### MUST

[Critical requirements that cannot be violated]

### MUST NOT

[Explicit prohibitions and anti-patterns to avoid]

## Examples

[2-3 concrete input/output pairs. Show simple cases first, then complex/edge cases. Use REAL data, not generic placeholders.]

### Example 1: [Simple/Common Case]

Input: [Actual example input]
Output: [Expected output]
Explanation: [Optional - only if non-obvious]

### Example 2: [Complex/Edge Case]

Input: [Actual example input]
Output: [Expected output]
Explanation: [Optional - only if non-obvious]

## Output Format

[Specification of expected output structure: JSON schema, template, markdown format, or structural requirements. Include data types where relevant.]

## Success Criteria

[Testable conditions to verify output correctness. Use checkboxes for clarity.]

- ✅ Criterion 1
- ✅ Criterion 2

## Edge Cases

[Known tricky situations and specific handling instructions. Only include if not covered in Examples.]
```

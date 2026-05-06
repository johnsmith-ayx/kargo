---
sidebar_label: json-write
description: Writes arbitrary data to a JSON file.
---

# `json-write`

`json-write` marshals arbitrary data and writes it to a JSON file. It can write
scalar values, objects, and arrays. If parent directories do not exist, they are
created automatically.

Unlike [`json-update`](json-update.md), this step does not preserve existing file
contents or formatting. It overwrites the target file.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to the JSON file to write. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `data` | `any` | Y | The value to marshal and write. Typically specified using an expression. Supports scalars, objects, and arrays. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the file written by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In this example, a YAML file is parsed into application objects, then each
object is written to a separate `config.json` file.

```yaml
steps:
- uses: yaml-parse
  as: src
  config:
    path: ./src/config/applications.yaml
    outputs:
    - name: frontend
      fromExpression: '.["frontend"]'
    - name: worker
      fromExpression: '.["worker"]'
    - name: api
      fromExpression: '.["api"]'

- uses: json-write
  config:
    path: ./out/apps/frontend/config.json
    data: ${{ outputs.src.frontend }}

- uses: json-write
  config:
    path: ./out/apps/worker/config.json
    data: ${{ outputs.src.worker }}

- uses: json-write
  config:
    path: ./out/apps/api/config.json
    data: ${{ outputs.src.api }}
```

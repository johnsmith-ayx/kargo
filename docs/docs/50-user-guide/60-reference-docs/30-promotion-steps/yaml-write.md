---
sidebar_label: yaml-write
description: Writes arbitrary data to a YAML file.
---

# `yaml-write`

`yaml-write` marshals arbitrary data and writes it to a YAML file. It can write
scalar values, objects, and arrays. If parent directories do not exist, they are
created automatically.

Unlike [`yaml-update`](yaml-update.md), this step does not preserve existing file
contents, comments, or formatting. It overwrites the target file.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | `string` | Y | Path to the YAML file to write. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `data` | `any` | Y | The value to marshal and write. Typically specified using an expression. Supports scalars, objects, and arrays. |

## Output

| Name | Type | Description |
|------|------|-------------|
| `commitMessage` | `string` | A description of the file written by this step. Typically, a subsequent [`git-commit` step](git-commit.md) will reference this output and aggregate this commit message fragment with others like it to build a comprehensive commit message that describes all changes. |

## Examples

### Common Usage

In this example, a YAML file is parsed into application objects, then each
object is written to a separate YAML file.

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

- uses: yaml-write
  config:
    path: ./out/apps/frontend/config.yaml
    data: ${{ outputs.src.frontend }}

- uses: yaml-write
  config:
    path: ./out/apps/worker/config.yaml
    data: ${{ outputs.src.worker }}

- uses: yaml-write
  config:
    path: ./out/apps/api/config.yaml
    data: ${{ outputs.src.api }}
```

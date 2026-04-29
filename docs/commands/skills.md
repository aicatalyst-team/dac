# skills

List and install DAC agent skills into a project.

Agent skills are local instructions that help coding agents create and modify DAC dashboards correctly. DAC currently ships a bundled `create-dashboard` skill for dashboard authoring.

`dac init` installs bundled skills automatically for new projects. Use `dac skills install` for existing projects or to refresh a skill with `--force`.

## List Skills

```shell
dac skills list
```

This prints the bundled skills and their install targets.

## Install Skills

```shell
dac skills install --dir .
```

By default, this installs every bundled DAC skill into the target project:

```text
.claude/skills/create-dashboard/SKILL.md
.codex/skills/create-dashboard -> ../../.claude/skills/create-dashboard
```

Claude gets the real skill file. Codex gets a symlink to the same skill directory so both agents use one shared copy.

Install a specific skill:

```shell
dac skills install create-dashboard --dir .
```

Overwrite an existing customized skill:

```shell
dac skills install create-dashboard --dir . --force
```

Restart your agent session after installing skills so the agent can discover the new local instructions.

## Flags

| Flag | Alias | Description |
|------|-------|-------------|
| `--dir` | `-d` | Target project directory |
| `--force` | `-f` | Overwrite existing skill files |

## Notes

- Installing skills does not change dashboard behavior at runtime.
- Skills are optional; they improve agent authoring and review workflows.
- Existing files are not overwritten unless `--force` is provided.

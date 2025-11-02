we are going to write a simple  tool which can be executed as restic-profile hooks. The tool should be a single binary with the CLI "restic-kit --config=/path/to/config.toml action action-options ...". the config file consists of several blocks, one per action. each action can take different arguments. The required config parameters can be read from environmel variables starting with RESTIC_HOOKS_* or the config file, where the env vars have higher prio. use sane defaults where applicable and raise an error for required but missing values.

For starters, the tool should have three actions:
- notify-email: send an email. The smtp credentials, subject and receiver should be read from config. the body from a file specified as cli argument.
- notify-http: perform a singe http GET request to an address defined in the config
- wait-online: try to reach a configurable URL for a given amount of time with exponential backup. exit with non-zero if the URL was not reached in time.

the tool should be written in golang. try to keep the architecture simple. Add unit and integration tests, an example config file a readme, an AGENTS.md and a MIT license file with me, Kilian Lackhove as license holder.

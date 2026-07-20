# Steampipe Config Generator

Manage your [Steampipe](https://steampipe.io/) AWS config files at scale!


## What is this?

*steampipe-config-generator* is tool that generates configuration files for [steampipe-aws-plugin](https://hub.steampipe.io/plugins/turbot/aws). These files are used by Steampipe AWS plugin to connect to your AWS Accounts and fetch the desired data.

We have created this tool to facilitate the creation and management of these files in organizations with multiple accounts.  
If you want more details about this, check our blog post: [Automate your Steampipe AWS configuration with AWS Organizations](https://unicrons.cloud/en/2024/10/18/automate-your-steampipe-aws-configuration-with-aws-organizations/)

![](./docs/flow.png)


## Features

- Automate generation of `.aws/credentials` and `.steampipe/config/aws.spc` for your AWS Organization.
- Create Steampipe connection *[aggregators](https://steampipe.io/docs/managing/connections#using-aggregators)* using your AWS Organization Accounts tags.
- Skip AWS Accounts based on their organizational units.
- Assume an IAM role to fech AWS Organizations information.


## Requirements

- Valid AWS credentials with the following IAM actions:
  ```json
  "organizations:ListAccounts",
  "organizations:ListParents",
  "organizations:ListTagsForResource"
  ```
- An AWS IAM Role deployed in all your AWS accounts with your required permissions for Steampipe.


## How to use it

```bash
./steampipe_config_generator --role my-org-role-name
```

If you are executing the tool inside an EC2 instance use `--credential Ec2InstanceMetadata` flag.
If you are executing the tool inside an ECS container use `--credential EcsContainer` flag.

Run `./steampipe_config_generator --help` for the full list of flags, and
`./steampipe_config_generator --version` to print the installed version.

> [!WARNING]
> **Breaking change:** starting with this release, flags use a double dash (`--role`) instead of a
> single dash (`-role`). Update any scripts that invoke this tool.


### Create Aggregators

The [aws_connections.tmpl](/code/templates/aws_connections.tmpl) template is used to generate the AWS connections files where you can add the needed *aggregators*.

To create an *aggregators* based on your AWS Account names. E.g: The following template will create an aggregator with all your AWS Accounts whose name begins with `Sandbox`:
```go
connection "aws_sandbox" {
  plugin      = "aws"
  type        = "aggregator"
  connections = ["aws_sandbox_*"]
}
```

> [!NOTE]
> All AWS Account names are normalized to lowercase. Spaces and hyphens are replaced by `_`.

To create an *aggregators* based on your AWS Accounts tags.
E.g: The following template will create an aggregator with all your AWS Accounts that contains the tag `team:engineering`:
```go
{{ $teamEng := index .Tags "team,engineering" }}
connection "aws_engineering_team" {
  plugin      = "aws"
  type        = "aggregator"
  connections = [{{- range $index, $name := $teamEng -}}{{ if $index }}, {{ end }}"aws_{{ $name }}"{{- end }}]
}
```

#### Multi-value tags

By default, a tag is matched by its exact value (`team=frontend` only matches `index .Tags "team,frontend"`).
If an account can belong to more than one group at once, use `--tagSplit` to split a tag's value on one
or more delimiter characters, per tag key. Each resulting value becomes its own aggregator group.

E.g. with an account tagged `team=frontend:backend` and:
```bash
--tagSplit="team=:"
```
that account is included in both `index .Tags "team,frontend"` and `index .Tags "team,backend"` — the
combined value `team,frontend:backend` is not registered, only the split values are.

`--tagSplit` takes a `key=delimiter[,delimiter...]` pair (repeatable for multiple keys), where each
comma-separated `delimiter` is a single character to split on, e.g. `--tagSplit="team=:,-"` splits the
`team` tag on `:` **or** `-`. Only tags listed in `--tagSplit` are affected — every other tag keeps
today's exact-match behavior unchanged. Valid delimiter characters are limited to AWS's supported tag
character set: `. : + = @ _ / -`.


## Versioning

This project follows [Semantic Versioning](https://semver.org/): breaking changes (to CLI flags
or to the Go library API in `generator/`) bump the major version, new features bump the minor
version, and fixes bump the patch version. See [CHANGELOG.md](./CHANGELOG.md) for what changed
in each release.


## Contribute

We welcome all contributors!

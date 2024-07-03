# Geppetto - go LLM and GPT3 specific prompting framework

#![Retro cybernetic puppetmaster controlling a pinocchio puppet that is working on a computer, retro mainframe aesthetic](geppetto.jpg)

Geppetto is a framework to run "actions" against LLMs (Large Language Models).
It is ultimately meant to become a rich framework to declaratively create 
chained LLM application, but is currently mostly a wrapper around a simple
prompting API.

The main abstraction currently presented is the concept of a prompt "command", based
around the [glazed](https://github.com/go-go-golems/glazed) command abstraction.

The base concept of a command is a function that takes a
set of input flags and arguments, and outputs structured data.
Using this as a central abstraction of steps
inside most LLM prompting applications, we can easily chain them together 
commands while being able to run them easily on the terminal.

## Pinocchio

It comes with a command line tool called `pinocchio` that can be used to interact 
with different prompting applications interactively or from the command line.

### Installation

To install the `pinocchio` command line tool with homebrew, run:

```bash
brew tap go-go-golems/go-go-go
brew install go-go-golems/go-go-go/pinocchio
```

To install the `pinocchio` command using apt-get, run:

```bash
echo "deb [trusted=yes] https://apt.fury.io/go-go-golems/ /" >> /etc/apt/sources.list.d/fury.list
apt-get update
apt-get install pinocchio
```

To install using `yum`, run:

```bash
echo "
[fury]
name=Gemfury Private Repo
baseurl=https://yum.fury.io/go-go-golems/
enabled=1
gpgcheck=0
" >> /etc/yum.repos.d/fury.repo
yum install pinocchio
```

To install using `go get`, run:

```bash
go get -u github.com/go-go-golems/geppetto/cmd/pinocchio
```

Finally, install by downloading the binaries straight from [github](https://github.com/go-go-golems/geppetto/releases).

## Usage

Configure pinocchio by storing your OpenAI API key in ~/.pinocchio/config.yaml. Furthermore,
you can configure one or more locations for geppetto commands.

```yaml
openai-api-key: XXXX
repositories:
  - /Users/manuel/code/pinocchio
  - /Users/manuel/.pinocchio/repository
```

You can then start using `pinocchio`:

```bash
❯ pinocchio examples test --print-prompt
Pretend you are a scientist. What is the age of you?

❯ pinocchio examples test               

As a scientist, I do not have an age.

❯ pinocchio examples test --pretend "100 year old explorer" --print-prompt
Pretend you are a 100 year old explorer. What is the age of you?

❯ pinocchio examples test --pretend "100 year old explorer"               

I am 100 years old.
```

Pinocchio comes with a selection of [demo prompts](https://github.com/go-go-golems/geppetto/tree/main/cmd/pinocchio/prompts/examples)
as an inspiration.

## Creating your own prompt

Creating your own prompt is easy. Create a yaml file in one of the configure repositories. 
The directory layout will be mapped to the command verb hierarchy. For example,
the file `~/.pinocchio/repository/prompts/examples/test.yaml` will be available as the command
`pinocchio examples test`.

A prompt description is a yaml file with the following structure, as shown for a prompt
that can be used to rewrite text in a certain style. After a short description, the
flags and arguments configure how what variables will be used to interpolate the prompt at
the bottom.

```yaml
name: command-name
short: Rewrite text in a certain style
flags:
  - name: author
    type: stringList
    help: Inspired by authors
    default:
      - L. Ron Hubbard
      - Isaac Asimov
      - Richard Bandler
      - Robert Anton Wilson
  - name: adjective
    type: stringList
    help: Style adjectives
    default:
      - esoteric
      - retro
      - technical
      - seventies hip
      - science fiction
  - name: style
    type: string
    help: Style
    default: in a style reminiscent of seventies and eighties computer manuals
  - name: instructions
    type: string
    help: Additional instructions
arguments:
  - name: body
    type: stringFromFile
    help: Paragraph to rewrite
    required: true
prompt: |
  Rewrite the following paragraph, 
  {{ if .style }}in the style of {{ .style }},{{ end }}
  {{ if .adjective }}so that it sounds {{ .adjective | join ", " }}, {{ end }}
  {{ if .author }}in the style of {{ .author | join ", " }}. {{ end }}
  Don't mention any authors names.

  ---
  {{ .body -}}
  ---

  {{ if .instructions }} {{ .instructions }} {{ end }}

  ---
```

## Creating aliases

In addition to prompts, you can define aliases, which are just shortcuts to other commands, with certain flags
prefilled. The resulting yaml file can be placed alongside other commands in one of the configured repositories.

```shell
❯ pinocchio examples test --pretend "100 year old explorer" --create-alias old-explorer \
   | tee ~/.pinochio/repository/prompts/examples/old-explorer.yaml
name: old-explorer
aliasFor: test
flags:
    pretend: 100 year old explorer

❯ pinocchio examples old-explorer
I am 100 years old.
```

## Contributing

This is GO GO GOLEMS playground, and GO GO GOLEMS don't accept contributions. 
The structure of the project will significantly change as we go forward, but
the core concept of a declarative prompting structure will stay the same,
and as such, you should be reasonably safe writing YAMLs to be used with pinocchio.

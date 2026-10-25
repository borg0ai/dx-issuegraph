---
title: "issuegraph completion"
description: "Generate the autocompletion script for the specified shell"
---

{/* AUTO-GENERATED: do not edit manually */}

Generated from `issuegraph help --doc completion`.

Generate the autocompletion script for issuegraph for the specified shell.
See each sub-command's help for details on how to use the generated script.


```
issuegraph completion [command]
```

## issuegraph completion bash

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source &lt;(issuegraph completion bash)

To load completions for every new session, execute once:

#### Linux:

	issuegraph completion bash &gt; /etc/bash_completion.d/issuegraph

#### macOS:

	issuegraph completion bash &gt; $(brew --prefix)/etc/bash_completion.d/issuegraph

You will need to start a new shell for this setup to take effect.


```
issuegraph completion bash
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

## issuegraph completion fish

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	issuegraph completion fish | source

To load completions for every new session, execute once:

	issuegraph completion fish &gt; ~/.config/fish/completions/issuegraph.fish

You will need to start a new shell for this setup to take effect.


```
issuegraph completion fish [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

## issuegraph completion powershell

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	issuegraph completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
issuegraph completion powershell [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

## issuegraph completion zsh

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" &gt;&gt; ~/.zshrc

To load completions in your current shell session:

	source &lt;(issuegraph completion zsh)

To load completions for every new session, execute once:

#### Linux:

	issuegraph completion zsh &gt; "$&#123;fpath[1]&#125;/_issuegraph"

#### macOS:

	issuegraph completion zsh &gt; $(brew --prefix)/share/zsh/site-functions/_issuegraph

You will need to start a new shell for this setup to take effect.


```
issuegraph completion zsh [flags]
```

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

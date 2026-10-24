# Curried on the flake's `self` so default.nix's `src = self;` keeps working
# outside the `pkgs.callPackage` auto-argument machinery (callPackage cannot
# supply a flake output on its own). flake.nix applies this as
# `import ./overlay.nix self`.
self: final: prev: {
  # Unwrapped Go build of the issuegraph binary. Exposed so downstream consumers can
  # override inputs (vendorHash, buildGoModule, etc.) via the standard
  # callPackage `.override` pattern, e.g.
  #
  #   final: prev: {
  #     issuegraph-unwrapped = prev.issuegraph-unwrapped.override (prevArgs: {
  #       buildGoModule = args:
  #         prevArgs.buildGoModule (args // { vendorHash = "..."; });
  #     });
  #   }
  issuegraph-unwrapped = final.callPackage ./default.nix {
    inherit self;
    buildGoModule = final.buildGo126Module;
  };

  # Wrap the unwrapped binary with shell completions and a `issuegraph` alias.
  issuegraph = final.stdenv.mkDerivation {
    pname = "issuegraph";
    version = final.issuegraph-unwrapped.version;

    phases = [ "installPhase" ];

    env.ISSUEGRAPH_DISABLE_METRICS = "1";
    env.ISSUEGRAPH_DISABLE_EVENT_FLUSH = "1";

    installPhase = ''
      mkdir -p $out/bin
      cp ${final.issuegraph-unwrapped}/bin/issuegraph $out/bin/issuegraph

      # Create 'issuegraph' alias symlink
      ln -s issuegraph $out/bin/issuegraph

      # Generate shell completions
      mkdir -p $out/share/fish/vendor_completions.d
      mkdir -p $out/share/bash-completion/completions
      mkdir -p $out/share/zsh/site-functions

      $out/bin/issuegraph completion fish > $out/share/fish/vendor_completions.d/issuegraph.fish
      $out/bin/issuegraph completion bash > $out/share/bash-completion/completions/issuegraph
      $out/bin/issuegraph completion zsh > $out/share/zsh/site-functions/_bd
    '';

    meta = final.issuegraph-unwrapped.meta;

    passthru = {
      # Per-shell completion outputs for users who only want one shell.
      fish-completions = final.runCommand "issuegraph-fish-completions" { } ''
        mkdir -p $out/share/fish/vendor_completions.d
        ln -s ${final.issuegraph}/share/fish/vendor_completions.d/issuegraph.fish $out/share/fish/vendor_completions.d/issuegraph.fish
      '';

      bash-completions = final.runCommand "issuegraph-bash-completions" { } ''
        mkdir -p $out/share/bash-completion/completions
        ln -s ${final.issuegraph}/share/bash-completion/completions/issuegraph $out/share/bash-completion/completions/issuegraph
      '';

      zsh-completions = final.runCommand "issuegraph-zsh-completions" { } ''
        mkdir -p $out/share/zsh/site-functions
        ln -s ${final.issuegraph}/share/zsh/site-functions/_bd $out/share/zsh/site-functions/_bd
      '';
    };
  };
}

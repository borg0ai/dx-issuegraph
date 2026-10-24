pkgs: {
  default = pkgs.issuegraph;
  issuegraph = pkgs.issuegraph;

  # Separate completion packages for users who only want specific shells.
  fish-completions = pkgs.issuegraph.passthru.fish-completions;
  bash-completions = pkgs.issuegraph.passthru.bash-completions;
  zsh-completions = pkgs.issuegraph.passthru.zsh-completions;
}

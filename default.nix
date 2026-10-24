{
  lib,
  self,
  buildGoModule,
  git,
  ...
}:
buildGoModule {
  pname = "issuegraph";
  version = "1.3.0";

  src = self;

  # Point to the main Go package
  subPackages = [ "modules/cli" ];
  tags = [ "gms_pure_go" ];
  doCheck = false;

  # proxyVendor avoids vendor/modules.txt consistency checks when the vendored
  # tree lags go.mod/go.sum.
  proxyVendor = true;
  vendorHash = "sha256-DQdauEx5g48Xbxrz4wGLx4vkrQoYQX3FUx+7N/Y6YV4=";

  # Match go.mod to the selected Nix Go toolchain. buildGoModule also builds
  # vendored dependencies in the Nix sandbox, where toolchain downloads are not
  # available.
  postPatch = ''
    goVer="$(go env GOVERSION | sed 's/^go//')"
    go mod edit -go="$goVer"
  '';

  env.GOTOOLCHAIN = "local";

  # Git is required for tests
  nativeBuildInputs = [ git ];

  meta = with lib; {
    description = "issuegraph (issuegraph) - An issue tracker designed for AI-supervised coding workflows";
    homepage = "https://github.com/borg0ai/dx-issuegraph";
    license = licenses.mit;
    mainProgram = "issuegraph";
    maintainers = [ ];
  };
}

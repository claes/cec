{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }: (flake-utils.lib.eachDefaultSystem (system: let
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.default = pkgs.mkShell {
      buildInputs = with pkgs; [
        go_1_21
        gotools
        golint
        go-tools
        golangci-lint
        gnumake
        protobuf
        protoc-gen-go
        gopls
        go-outline
        gotools
        godef
        delve
        mqttui
        libcec
        libcec_platform
        pkg-config
      ];
    };
  }));
}

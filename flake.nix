{
  description = "WIP development shell for the ShotGlass GTK4 launcher";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gtk4
              gobject-introspection
              pkg-config
              gcc
            ];

            shellHook = ''
              echo "ShotGlass WIP development shell"
              echo "Run: go test ./..."
              echo "Run: go run ./cmd/shotglass"
            '';
          };
        });
    };
}

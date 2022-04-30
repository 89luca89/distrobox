{
  description = "A flake for distrobox";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # System types to support.
      supportedSystems = [ "x86_64-linux" "aarch64-linux"  ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; overlays = [ self.overlay ]; });

    in

    {

      # A Nixpkgs overlay.
      overlay = final: prev: {

        distrobox = with final; stdenv.mkDerivation rec {
          name = "distrobox-${version}";
          src = ./.;
          installPhase =
            ''
              runHook preInstall

              mkdir -p $out/bin 
              #./install -p $out/bin 
              install -Dm755 -t "$out/bin" distrobox distrobox-create distrobox-enter distrobox-ephemeral distrobox-export distrobox-host-exec distrobox-init distrobox-rm distrobox-stop

              runHook postInstall
            '';
        };

      };

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        {
          inherit (nixpkgsFor.${system}) distrobox;
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.distrobox);


      # Tests run by 'nix flake check' and by Hydra.
      checks = forAllSystems
        (system:
          with nixpkgsFor.${system};

          {
            inherit (self.packages.${system}) distrobox;

            # Additional tests, if applicable.
            test = stdenv.mkDerivation {
              name = "distrobox-test-${version}";

              buildInputs = [ distrobox ];

              unpackPhase = "true";

              buildPhase = ''
                echo 'running some integration tests'
                podman rm -f my-distrobox
                if distrobox create; then
                  echo 'distrobox create is functioning'
                else
                  echo 'distrobox create is not functioning'
                  exit 1
                fi
                if distrobox enter
              '';

              installPhase = "mkdir -p $out";
            };
          }

          // lib.optionalAttrs stdenv.isLinux {
            # A VM test of the NixOS module.
            vmTest =
              with import (nixpkgs + "/nixos/lib/testing-python.nix") {
                inherit system;
              };

              makeTest {
                nodes = {
                  client = { ... }: {
                    imports = [ self.nixosModules.hello ];
                  };
                };

                testScript =
                  ''
                    start_all()
                    client.wait_for_unit("multi-user.target")
                    client.succeed("hello")
                  '';
              };
          }
        );

    };
}

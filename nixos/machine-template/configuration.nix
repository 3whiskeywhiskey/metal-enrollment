{ config, pkgs, lib, modulesPath, ... }:

# This is a template configuration for enrolled machines
# The Metal Enrollment system will customize this based on
# the configuration provided in the web dashboard

{
  imports = [
    "${modulesPath}/installer/netboot/netboot-minimal.nix"
  ];

  # System configuration
  boot.kernelPackages = pkgs.linuxPackages_latest;
  boot.kernelParams = [ "console=ttyS0,115200" "console=tty0" ];

  # Network configuration
  networking.hostName = "HOSTNAME_PLACEHOLDER";
  networking.useDHCP = true;
  networking.firewall.enable = true;

  # Enable SSH
  services.openssh = {
    enable = true;
    settings = {
      PermitRootLogin = "prohibit-password";
      PasswordAuthentication = false;
    };
  };

  # Add your SSH keys here
  users.users.root.openssh.authorizedKeys.keys = [
    # "ssh-rsa AAAA... user@example.com"
  ];

  # System packages
  environment.systemPackages = with pkgs; [
    vim
    git
    htop
    curl
    wget
  ];

  # Enable serial console
  systemd.services."serial-getty@ttyS0".enable = true;

  system.stateVersion = "24.05";

  # Custom configuration will be inserted here by the enrollment system
  # CUSTOM_CONFIG_PLACEHOLDER
}

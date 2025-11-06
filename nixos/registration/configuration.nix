{ config, pkgs, lib, modulesPath, ... }:

{
  imports = [
    "${modulesPath}/installer/netboot/netboot-minimal.nix"
  ];

  # Kernel and boot configuration
  boot.kernelPackages = pkgs.linuxPackages_latest;
  boot.kernelParams = [ "console=ttyS0,115200" "console=tty0" ];

  # Enable serial console
  boot.loader.grub.extraConfig = ''
    serial --unit=0 --speed=115200
    terminal_input serial console
    terminal_output serial console
  '';

  # Network configuration
  networking.hostName = "metal-enrollment-registration";
  networking.useDHCP = true;
  networking.firewall.enable = false;

  # Essential packages for hardware detection
  environment.systemPackages = with pkgs; [
    pciutils
    usbutils
    dmidecode
    lshw
    hdparm
    smartmontools
    ethtool
    ipmitool
    curl
    jq
    bash
  ];

  # Hardware detection and enrollment script
  systemd.services.metal-enrollment = {
    description = "Metal Enrollment Registration Service";
    wantedBy = [ "multi-user.target" ];
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];

    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = false;
      ExecStart = "${pkgs.bash}/bin/bash /etc/metal-enrollment/enroll.sh";
    };
  };

  # Enrollment script
  environment.etc."metal-enrollment/enroll.sh" = {
    text = builtins.readFile ./enroll.sh;
    mode = "0755";
  };

  # Auto-login on console for debugging
  services.getty.autologinUser = "root";

  # Enable SSH for remote access (debugging)
  services.openssh = {
    enable = true;
    settings = {
      PermitRootLogin = "yes";
      PasswordAuthentication = false;
    };
  };

  # Set a default root password for console access (change this!)
  users.users.root.initialPassword = "enrollment";

  # Minimal system state version
  system.stateVersion = "24.05";
}

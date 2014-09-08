# -*- mode: ruby -*-
# vi: set ft=ruby :

$PROVISION = <<SCRIPT
echo I am but an empty shell
apt-get -y install nsca
sed -i "s/#password=/password=password/" /etc/nsca.cfg
sed -i "s/decryption_method=1/decryption_method=2/" /etc/nsca.cfg
sed -i "s/debug=0/debug=1/" /etc/nsca.cfg
/etc/init.d/nsca restart
SCRIPT

Vagrant::Config.run do |config|
  config.vm.box = "nsca"
  config.vm.box_url = "https://jimdo-vagrant-boxes.s3.amazonaws.com/jimdo-debian-6.0.7.box"
  config.vm.forward_port 5667, 5667
  config.vm.provision "shell", inline: $PROVISION
end

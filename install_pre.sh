sudo dnf install perl perl-core perl-App-cpanminus perl-CPAN
sudo dnf install zlib-devel
sudo dnf config-manager --set-enabled crb && sudo dnf install -y libyaml-devel
class Bandit < Formula
  desc "Autonomous local terminal AI agent powered by Go and Rust"
  homepage "https://github.com/NichuSPN/bandit"
  version "1.0.0"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/NichuSPN/bandit/releases/download/v1.0.0/bandit-v1.0.0-darwin-arm64.tar.gz"
    sha256 "REPLACE_WITH_DARWIN_ARM64_SHA256"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/NichuSPN/bandit/releases/download/v1.0.0/bandit-v1.0.0-darwin-amd64.tar.gz"
    sha256 "REPLACE_WITH_DARWIN_AMD64_SHA256"
  elsif OS.linux?
    url "https://github.com/NichuSPN/bandit/releases/download/v1.0.0/bandit-v1.0.0-linux-amd64.tar.gz"
    sha256 "REPLACE_WITH_LINUX_AMD64_SHA256"
  end

  def install
    bin.install "bandit"
  end

  test do
    assert_match "Bandit", shell_output("#{bin}/bandit --help")
  end
end

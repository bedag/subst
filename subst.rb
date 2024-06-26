# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Subst < Formula
  desc ""
  homepage ""
  version "0.0.1-alpha9"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/bedag/subst/releases/download/v0.0.1-alpha9/subst_0.0.1-alpha9_darwin_amd64.tar.gz"
      sha256 "dfd31dc2e0f2373baf52732a2674bb45b297f4beb2aa3111847e48b191506e05"

      def install
        bin.install "subst"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/bedag/subst/releases/download/v0.0.1-alpha9/subst_0.0.1-alpha9_darwin_arm64.tar.gz"
      sha256 "8407cb57e0457d95ebbd005afdff05aa060a13112223773dcdda5400b3013755"

      def install
        bin.install "subst"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/bedag/subst/releases/download/v0.0.1-alpha9/subst_0.0.1-alpha9_linux_arm64.tar.gz"
      sha256 "718eb07b606178f731d803dddd5c938bbddf53fe12c4f56212cfc2d81650a1a5"

      def install
        bin.install "subst"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/bedag/subst/releases/download/v0.0.1-alpha9/subst_0.0.1-alpha9_linux_amd64.tar.gz"
      sha256 "36fb49ca08918c2e117de8431fa3e6650406ec94393b80add18e71b4d91b8d12"

      def install
        bin.install "subst"
      end
    end
    if Hardware::CPU.arm? && !Hardware::CPU.is_64_bit?
      url "https://github.com/bedag/subst/releases/download/v0.0.1-alpha9/subst_0.0.1-alpha9_linux_armv6.tar.gz"
      sha256 "7d4e84438f9d1442a99bbd9cef161f8f641ffe67174ea1bc47dcba8098f1ad26"

      def install
        bin.install "subst"
      end
    end
  end
end

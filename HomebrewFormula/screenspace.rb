cask "screenspace" do
  version "0.1.0"
  sha256 :no_check

  url "https://github.com/0x63616c/screenspace/releases/download/v#{version}/ScreenSpace.dmg"
  name "ScreenSpace"
  desc "Open-source live wallpaper app for macOS"
  homepage "https://github.com/0x63616c/screenspace"

  depends_on macos: ">= :sequoia"

  app "ScreenSpace.app"

  zap trash: [
    "~/Library/Application Support/ScreenSpace",
    "~/Library/Screen Savers/ScreenSpaceSaver.saver",
  ]
end

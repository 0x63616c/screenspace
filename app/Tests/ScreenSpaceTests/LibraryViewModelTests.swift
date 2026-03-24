import Foundation
import Testing
@testable import ScreenSpace

@MainActor
struct LibraryViewModelTests {
    @Test("removeVideo removes from list and filesystem")
    func removeVideoDeletesFile() {
        let fs = MockFileSystem()
        let provider = MockWallpaperProvider()
        let url = URL(fileURLWithPath: "/tmp/test.mp4")
        fs.files[url.path] = Data()
        let vm = LibraryViewModel(fileSystem: fs, wallpaperProvider: provider, eventLog: MockEventLog())
        vm.localVideos = [url]

        vm.removeVideo(url: url)

        #expect(vm.localVideos.isEmpty)
        #expect(fs.files[url.path] == nil)
    }

    @Test("setWallpaper calls wallpaper provider and logs event")
    func setWallpaperCallsProvider() {
        let provider = MockWallpaperProvider()
        let log = MockEventLog()
        let vm = LibraryViewModel(fileSystem: MockFileSystem(), wallpaperProvider: provider, eventLog: log)
        let url = URL(fileURLWithPath: "/tmp/video.mp4")

        vm.setWallpaper(url: url)

        #expect(!provider.setCalls.isEmpty)
        #expect(provider.setCalls.first?.url == url)
        #expect(log.events.contains { $0.event == "wallpaper_set" })
    }

    @Test("handleDroppedURLs skips invalid video files")
    func skipsInvalidFiles() {
        let vm = LibraryViewModel(
            fileSystem: MockFileSystem(),
            wallpaperProvider: MockWallpaperProvider(),
            eventLog: MockEventLog()
        )
        let invalid = URL(fileURLWithPath: "/tmp/document.pdf")

        vm.handleDroppedURLs([invalid])

        #expect(vm.localVideos.isEmpty)
        #expect(vm.importError == nil)
    }
}

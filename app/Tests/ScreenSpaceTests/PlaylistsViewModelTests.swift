import Testing
@testable import ScreenSpace

@MainActor
struct PlaylistsViewModelTests {
    @Test("load populates playlists from manager")
    func loadPopulates() {
        let manager = MockPlaylistManager()
        manager.existingPlaylists = [TestFixtures.playlist(id: "p1"), TestFixtures.playlist(id: "p2")]
        let vm = PlaylistsViewModel(playlistManager: manager, eventLog: MockEventLog())

        vm.load()

        #expect(vm.playlists.count == 2)
    }

    @Test("create adds playlist with trimmed name")
    func createAddsPlaylist() {
        let manager = MockPlaylistManager()
        let vm = PlaylistsViewModel(playlistManager: manager, eventLog: MockEventLog())
        vm.newPlaylistName = "  Nature  "

        vm.create()

        #expect(manager.createdNames == ["Nature"])
        #expect(vm.newPlaylistName.isEmpty)
    }

    @Test("create does nothing with empty name")
    func createIgnoresEmptyName() {
        let manager = MockPlaylistManager()
        let vm = PlaylistsViewModel(playlistManager: manager, eventLog: MockEventLog())
        vm.newPlaylistName = ""

        vm.create()

        #expect(manager.createdNames.isEmpty)
    }

    @Test("updateShuffle persists change")
    func updateShufflePersists() {
        let manager = MockPlaylistManager()
        let playlist = TestFixtures.playlist(id: "p1")
        manager.existingPlaylists = [playlist]
        let vm = PlaylistsViewModel(playlistManager: manager, eventLog: MockEventLog())
        vm.load()

        vm.updateShuffle(playlist, enabled: true)

        #expect(manager.updatedPlaylists.first?.shuffle == true)
    }

    @Test("delete removes playlist and reloads")
    func deleteRemovesPlaylist() {
        let manager = MockPlaylistManager()
        let playlist = TestFixtures.playlist(id: "p1")
        manager.existingPlaylists = [playlist]
        let log = MockEventLog()
        let vm = PlaylistsViewModel(playlistManager: manager, eventLog: log)
        vm.load()

        vm.delete(playlist: playlist)

        #expect(manager.deletedIDs.contains("p1"))
        #expect(log.events.contains { $0.event == "playlist_advanced" })
    }
}

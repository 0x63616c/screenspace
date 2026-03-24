import Foundation
import Testing
@testable import ScreenSpace

@MainActor
struct UploadViewModelTests {
    @Test("canUpload is false with no file selected")
    func canUploadFalseNoFile() {
        let vm = UploadViewModel(api: MockAPI(), fileSystem: MockFileSystem(), eventLog: MockEventLog())
        vm.title = "Test"
        vm.acceptedPolicy = true

        #expect(vm.canUpload == false)
    }

    @Test("canUpload is false with empty title")
    func canUploadFalseEmptyTitle() {
        let vm = UploadViewModel(api: MockAPI(), fileSystem: MockFileSystem(), eventLog: MockEventLog())
        vm.selectedFileURL = URL(fileURLWithPath: "/tmp/test.mp4")
        vm.acceptedPolicy = true
        vm.title = "   "

        #expect(vm.canUpload == false)
    }

    @Test("canUpload is false without policy acceptance")
    func canUploadFalseNoPolicy() {
        let vm = UploadViewModel(api: MockAPI(), fileSystem: MockFileSystem(), eventLog: MockEventLog())
        vm.selectedFileURL = URL(fileURLWithPath: "/tmp/test.mp4")
        vm.title = "Test"
        vm.acceptedPolicy = false

        #expect(vm.canUpload == false)
    }

    @Test("canUpload is true when all conditions met")
    func canUploadTrue() {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/test.mp4")
        fs.files[url.path] = Data(count: 50_000_000)
        let vm = UploadViewModel(api: MockAPI(), fileSystem: fs, eventLog: MockEventLog())
        vm.selectedFileURL = url
        vm.title = "Test Wallpaper"
        vm.acceptedPolicy = true

        #expect(vm.canUpload == true)
        #expect(vm.fileTooLarge == false)
    }

    @Test("fileTooLarge is true for files over 200MB")
    func fileTooLargeDetected() {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/big.mp4")
        fs.files[url.path] = Data(count: 250_000_000)
        let vm = UploadViewModel(api: MockAPI(), fileSystem: fs, eventLog: MockEventLog())
        vm.selectedFileURL = url

        #expect(vm.fileTooLarge == true)
    }

    @Test("parsedTags splits comma-separated input")
    func parsedTagsSplitsInput() {
        let vm = UploadViewModel(api: MockAPI(), fileSystem: MockFileSystem(), eventLog: MockEventLog())
        vm.tagsText = "ocean, waves, nature"

        #expect(vm.parsedTags == ["ocean", "waves", "nature"])
    }

    @Test("parsedTags ignores empty segments")
    func parsedTagsIgnoresEmpty() {
        let vm = UploadViewModel(api: MockAPI(), fileSystem: MockFileSystem(), eventLog: MockEventLog())
        vm.tagsText = "ocean,,waves"

        #expect(vm.parsedTags == ["ocean", "waves"])
    }

    @Test("loadCategories falls back to all on error")
    func loadCategoriesFallback() async {
        let api = MockAPI()
        api.categoriesResponse = .failure(APIError.httpError(status: 503))
        let vm = UploadViewModel(api: api, fileSystem: MockFileSystem(), eventLog: MockEventLog())

        await vm.loadCategories()

        #expect(vm.categories == Category.allCases)
    }
}

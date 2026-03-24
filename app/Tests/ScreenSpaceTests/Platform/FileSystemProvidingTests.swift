import Testing
import Foundation
@testable import ScreenSpace

@Suite("MockFileSystem")
@MainActor
struct FileSystemProvidingTests {
    @Test("write stores data and fileExists returns true")
    func writeAndExists() throws {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/test.txt")
        let data = Data("hello".utf8)
        try fs.write(data: data, to: url)
        #expect(fs.fileExists(at: url))
    }

    @Test("fileExists returns false for missing file")
    func missingFile() {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/missing.txt")
        #expect(!fs.fileExists(at: url))
    }

    @Test("remove deletes file")
    func removeFile() throws {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/test.txt")
        try fs.write(data: Data("hello".utf8), to: url)
        try fs.remove(at: url)
        #expect(!fs.fileExists(at: url))
    }

    @Test("contentsOfDirectory returns files in directory")
    func contentsOfDirectory() throws {
        let fs = MockFileSystem()
        let dir = URL(fileURLWithPath: "/tmp/mydir")
        let file1 = URL(fileURLWithPath: "/tmp/mydir/a.txt")
        let file2 = URL(fileURLWithPath: "/tmp/mydir/b.txt")
        try fs.write(data: Data("a".utf8), to: file1)
        try fs.write(data: Data("b".utf8), to: file2)
        let contents = try fs.contentsOfDirectory(at: dir)
        #expect(contents.count == 2)
    }

    @Test("fileSize returns data length")
    func fileSize() throws {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/test.txt")
        let data = Data("hello".utf8)
        try fs.write(data: data, to: url)
        let size = try fs.fileSize(at: url)
        #expect(size == Int64(data.count))
    }

    @Test("createDirectory adds to directories set")
    func createDirectory() throws {
        let fs = MockFileSystem()
        let url = URL(fileURLWithPath: "/tmp/newdir")
        try fs.createDirectory(at: url)
        #expect(fs.fileExists(at: url))
    }
}

import Testing
@testable import ScreenSpace

@Suite("MockKeychain")
@MainActor
struct KeychainProvidingTests {
    @Test("saves and loads data by key")
    func saveAndLoad() throws {
        let keychain = MockKeychain()
        let data = Data("token".utf8)
        try keychain.save(key: "auth_token", data: data)
        let loaded = keychain.load(key: "auth_token")
        #expect(loaded == data)
    }

    @Test("returns nil for missing key")
    func missingKey() {
        let keychain = MockKeychain()
        #expect(keychain.load(key: "missing") == nil)
    }

    @Test("delete removes key")
    func deleteKey() throws {
        let keychain = MockKeychain()
        try keychain.save(key: "auth_token", data: Data("x".utf8))
        keychain.delete(key: "auth_token")
        #expect(keychain.load(key: "auth_token") == nil)
    }

    @Test("overwrite replaces existing value")
    func overwrite() throws {
        let keychain = MockKeychain()
        try keychain.save(key: "k", data: Data("v1".utf8))
        try keychain.save(key: "k", data: Data("v2".utf8))
        #expect(keychain.load(key: "k") == Data("v2".utf8))
    }
}

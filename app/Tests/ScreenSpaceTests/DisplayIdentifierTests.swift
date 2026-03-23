import XCTest
@testable import ScreenSpace

final class DisplayIdentifierTests: XCTestCase {
    func testStableIDFromScreen() throws {
        guard let screen = NSScreen.main else {
            throw XCTSkip("No screen available")
        }
        let id = DisplayIdentifier.stableID(for: screen)
        XCTAssertFalse(id.isEmpty)

        let id2 = DisplayIdentifier.stableID(for: screen)
        XCTAssertEqual(id, id2)
    }
}

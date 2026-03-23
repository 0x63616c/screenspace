import XCTest
@testable import ScreenSpace

final class APIClientTests: XCTestCase {
    func testBuildURL() {
        let client = APIClient(baseURL: "https://api.screenspace.app")
        let url = client.buildURL(path: "/api/v1/wallpapers", query: ["sort": "popular", "limit": "20"])
        XCTAssertNotNil(url)

        let components = URLComponents(url: url!, resolvingAgainstBaseURL: false)
        let queryItems = Set(components?.queryItems ?? [])
        let expected = Set([URLQueryItem(name: "limit", value: "20"), URLQueryItem(name: "sort", value: "popular")])
        XCTAssertEqual(queryItems, expected)
    }

    func testBuildURLNoQuery() {
        let client = APIClient(baseURL: "https://api.screenspace.app")
        let url = client.buildURL(path: "/api/v1/wallpapers/abc")
        XCTAssertEqual(url?.absoluteString, "https://api.screenspace.app/api/v1/wallpapers/abc")
    }

    func testDecodeWallpaperResponse() throws {
        let json = """
        {"id":"abc","title":"Ocean","resolution":"3840x2160","width":3840,"height":2160,
         "duration":30.0,"file_size":85000000,"format":"h264","download_count":100,
         "category":"nature","tags":["ocean","waves"]}
        """.data(using: .utf8)!

        let wallpaper = try JSONDecoder().decode(WallpaperResponse.self, from: json)
        XCTAssertEqual(wallpaper.title, "Ocean")
        XCTAssertEqual(wallpaper.width, 3840)
        XCTAssertEqual(wallpaper.fileSize, 85000000)
        XCTAssertEqual(wallpaper.category, "nature")
    }

    func testDecodeAuthResponse() throws {
        let json = """
        {"token":"jwt-token-here","role":"admin"}
        """.data(using: .utf8)!

        let auth = try JSONDecoder().decode(AuthResponse.self, from: json)
        XCTAssertEqual(auth.token, "jwt-token-here")
        XCTAssertEqual(auth.role, "admin")
    }
}

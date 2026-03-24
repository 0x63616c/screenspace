// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "ScreenSpace",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(name: "ScreenSpace", targets: ["ScreenSpace"])
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-openapi-generator", from: "1.6.0"),
        .package(url: "https://github.com/apple/swift-openapi-runtime", from: "1.7.0"),
        .package(url: "https://github.com/apple/swift-openapi-urlsession", from: "1.0.0")
    ],
    targets: [
        .target(
            name: "ScreenSpaceAPI",
            dependencies: [
                .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
                .product(name: "OpenAPIURLSession", package: "swift-openapi-urlsession")
            ],
            path: "Sources/ScreenSpaceAPI",
            plugins: [
                .plugin(name: "OpenAPIGenerator", package: "swift-openapi-generator")
            ]
        ),
        .executableTarget(
            name: "ScreenSpace",
            dependencies: [
                "ScreenSpaceAPI",
                .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
                .product(name: "OpenAPIURLSession", package: "swift-openapi-urlsession")
            ],
            path: "Sources/ScreenSpace"
        ),
        .testTarget(
            name: "ScreenSpaceTests",
            dependencies: ["ScreenSpace"],
            path: "Tests/ScreenSpaceTests"
        )
    ]
)

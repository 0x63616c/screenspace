// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "ScreenSpace",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(name: "ScreenSpace", targets: ["ScreenSpace"]),
    ],
    dependencies: [
        // Sparkle will be added later
    ],
    targets: [
        .executableTarget(
            name: "ScreenSpace",
            dependencies: [],
            path: "Sources/ScreenSpace"
        ),
        .testTarget(
            name: "ScreenSpaceTests",
            dependencies: ["ScreenSpace"],
            path: "Tests/ScreenSpaceTests"
        ),
    ]
)

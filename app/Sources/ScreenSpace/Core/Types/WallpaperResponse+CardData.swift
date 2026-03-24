import Foundation

extension WallpaperResponse {
    func toCardData() -> WallpaperCardData {
        WallpaperCardData(
            id: id,
            title: title,
            thumbnailURL: thumbnailURL.flatMap { URL(string: $0) },
            width: width,
            height: height,
            duration: duration
        )
    }
}

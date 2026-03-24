import Foundation
import ScreenSpaceAPI

extension Components.Schemas.Wallpaper {
    func toCardData() -> WallpaperCardData {
        WallpaperCardData(
            id: id,
            title: title,
            thumbnailURL: URL(string: thumbnailUrl),
            width: width,
            height: height,
            duration: duration
        )
    }
}

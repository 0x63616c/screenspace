import Foundation

func formatFileSize(_ bytes: Int64) -> String {
    let mb = Double(bytes) / 1_000_000
    return String(format: "%.1f MB", mb)
}

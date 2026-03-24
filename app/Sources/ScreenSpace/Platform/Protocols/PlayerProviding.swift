import Foundation

protocol PlayerProviding: Sendable {
    func play(url: URL)
    func pause()
    func resume()
    func seek(to time: Double)
    func stop()
}

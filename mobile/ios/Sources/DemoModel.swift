import Connect
import Foundation
import SwiftProtobuf
import SwiftUI

enum NativeAPI {
    static let host = ProcessInfo.processInfo.environment["RPC_BASE_URL"] ?? "http://localhost:3000"
    static func client(host: String = host) -> App_V1_AppServiceClient {
        App_V1_AppServiceClient(client: ProtocolClient(config: ProtocolClientConfig(
            host: host, networkProtocol: .connect, codec: ProtoCodec(), timeout: 5
        )))
    }
}

@MainActor
final class DemoModel: ObservableObject {
    enum Output {
        case page(App_V1_GetIosUserPageResponse)
        case summary(App_V1_GetIosUserSummaryResponse)
    }
    @Published var userID = "u1"
    @Published var summary = false
    @Published private(set) var loading = false
    @Published private(set) var output: Output?
    @Published private(set) var json = "API を呼び出してください。"
    @Published private(set) var error = ""
    @Published private(set) var method = ""
    private let client = NativeAPI.client()

    func load() async {
        loading = true
        output = nil
        error = ""
        defer { loading = false }
        do {
            if summary {
                method = "GetIosUserSummary"
                let response = await client.getIosUserSummary(request: .with { $0.id = userID })
                let message = try response.result.get()
                output = .summary(message)
                json = try message.jsonString()
            } else {
                method = "GetIosUserPage"
                let response = await client.getIosUserPage(request: .with { $0.id = userID })
                let message = try response.result.get()
                output = .page(message)
                json = try message.jsonString()
            }
        } catch {
            self.error = String(describing: error)
            json = ""
        }
    }
}

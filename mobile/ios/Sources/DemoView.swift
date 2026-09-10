import SwiftUI

struct DemoView: View {
    @StateObject private var model = DemoModel()
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    Text("フラグメントから、\nネイティブの RPC へ。")
                        .font(.system(size: 29, weight: .semibold))
                    Text("SwiftUI → 専用 Protobuf API → gRPC services")
                        .font(.caption).foregroundStyle(.secondary)
                    VStack(alignment: .leading, spacing: 16) {
                        Picker("取得データ", selection: $model.summary) {
                            Text("プロフィール").tag(false)
                            Text("サマリー").tag(true)
                        }.pickerStyle(.segmented).accessibilityIdentifier("operation")
                        HStack {
                            ForEach(["u1", "u2", "u3", "missing"], id: \.self) { id in
                                Button(id) { model.userID = id }
                                    .buttonStyle(.bordered)
                                    .tint(model.userID == id ? .indigo : .gray)
                                    .accessibilityIdentifier("user-\(id)")
                            }
                        }
                        Button { Task { await model.load() } } label: {
                            HStack { Spacer(); Text(model.loading ? "取得中…" : "生成 API を呼び出す"); Spacer() }
                        }.buttonStyle(.borderedProminent).tint(.indigo)
                            .accessibilityIdentifier("fetch")
                    }.disabled(model.loading)
                    if !model.error.isEmpty {
                        Text(model.error).foregroundStyle(.red).accessibilityIdentifier("rpc-error")
                    }
                    if let output = model.output {
                        switch output {
                        case .page(let page): IosUserPage(data: page)
                        case .summary(let summary): IosUserSummary(data: summary)
                        }
                    }
                    VStack(alignment: .leading, spacing: 12) {
                        Text(model.method.isEmpty ? "生成されたレスポンス" : model.method)
                            .font(.caption.bold()).accessibilityIdentifier("rpc-method")
                        Text(model.json).font(.system(size: 11, design: .monospaced))
                            .textSelection(.enabled).accessibilityIdentifier("rpc-json")
                    }.padding().frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color(red: 0.14, green: 0.20, blue: 0.31))
                        .foregroundStyle(.white).clipShape(RoundedRectangle(cornerRadius: 12))
                    Text("View と同じファイルの宣言をビルド時に合成。\n通信には生成済みの Swift クライアントを使用。")
                        .font(.caption).foregroundStyle(.secondary)
                }.padding(22)
            }.background(Color(red: 0.94, green: 0.96, blue: 0.99))
                .navigationTitle("Fragment RPC").navigationBarTitleDisplayMode(.inline)
        }
    }
}

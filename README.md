# Fragment RPC PoC

A proof of concept for colocating GraphQL fragments with React, SwiftUI, and Jetpack Compose components, then generating typed Protobuf RPCs at build time. Clients send variables over Connect; the server calls local gRPC backends without a GraphQL runtime.

## Setup

Requires an Apple Silicon Mac, Node.js 22.14+ (22.x recommended), npm, `protoc`, Xcode 26 with Swift 6.2 and an iOS Simulator, and XcodeGen.

Generation collects all three clients, so the native tooling is required even when running only the web demo.

```sh
npm ci
bash scripts/setup-swift.sh
bash scripts/setup-mobile.sh
bash scripts/build-native-tools.sh
```

The setup scripts download SDKs and build tools into `.local/mobile-tools`. Accept the Android SDK licenses when prompted.

## Run

```sh
npm run dev
```

Open [the web demo](http://127.0.0.1:3000/). The server includes fictional users and organizations. Compare profile and summary requests, or select a user without an organization or a missing user.

Keep the server running while using the native apps. Run each command below in a separate terminal:

```sh
bash scripts/run-ios.sh
bash scripts/start-android-emulator.sh --window
bash scripts/run-android.sh
```

Wait for the Android emulator to boot before running `run-android.sh`. The scripts build, install, and launch the apps. Open Simulator to view the iOS app.

The default targets are `iPhone 17` and `emulator-5556`; override them with `IOS_SIMULATOR` and `ANDROID_SERIAL`. The API uses port 3000, and the gRPC endpoint uses port 50051. Run `npm run demo` to try the gRPC client directly.

## Development

Web fragments live beside components in `.graphql` files. Native fragments are embedded in the component's Swift or Kotlin file using `@GraphQLFragment` and `@GraphQLQuery` declarations.

After changing a fragment, restart `npm run dev` and rebuild the relevant native app. There is no hot reload. Use `npm run generate` for generation alone; rebuild the extraction tools with `bash scripts/build-native-tools.sh` after changing them.

Keep `generated/schema.lock.json` under version control alongside the generated contracts. It preserves Protobuf field numbers across changes; do not delete or reset it.

## Test

```sh
npm run check
swift test --package-path mobile/swift-graphql
npm run test:ios
npm run test:android
```

The native tests require the local API and their simulator or emulator to be running. They cover API calls and basic UI flows.

## Limitations

- Local demonstration with fictional data. Physical devices, production deployment, authentication, and TLS are outside the scope.
- Supports a subset of GraphQL queries and fragments. Mutations, subscriptions, polymorphic types, dynamic embedded documents, full Relay fragment masking, and normalized caching are not implemented.
- Android fragment adapters currently generate incorrect accessors for list fields. The supplied demo does not use lists.

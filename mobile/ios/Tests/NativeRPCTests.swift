import Connect
import Foundation
import XCTest
@testable import FragmentRPC

final class NativeRPCTests: XCTestCase {
    func testGeneratedProfileAndSummaryAgainstLiveGateway() async throws {
        let client = NativeAPI.client()
        let profile = try await client.getIosUserPage(request: .with { $0.id = "u1" }).result.get()
        XCTAssertTrue(profile.hasUser)
        XCTAssertEqual(profile.user.name, "Aki Tanaka")
        XCTAssertEqual(profile.user.organization.name, "Northstar Studio")
        XCTAssertEqual(profile.user.avatarURL, "/avatars/aki.svg")
        let card: IosUserCardFragment.Data = profile.user.asIosUserCardFragmentData()
        let avatar: IosAvatarFragment.Data = card.asIosAvatarFragmentData()
        XCTAssertEqual(avatar.avatarUrl, "/avatars/aki.svg")
        XCTAssertEqual(card.organization?.asIosOrganizationBadgeFragmentData().name, "Northstar Studio")
        XCTAssertEqual(Mirror(reflecting: avatar).children.compactMap(\.label), ["avatarUrl"])
        XCTAssertTrue(IosUserCardFragment.document.contains("...IosAvatar_user"))
        let json = try JSONSerialization.jsonObject(with: profile.jsonUTF8Data()) as! [String: Any]
        XCTAssertNil((json["user"] as! [String: Any])["email"])
        let summary = try await client.getIosUserSummary(request: .with { $0.id = "u2" }).result.get()
        XCTAssertEqual(summary.user.name, "Ren Sato")
        XCTAssertFalse(try summary.jsonString().contains("organization"))
    }

    func testPresenceAndMissingRequiredInput() async throws {
        let client = NativeAPI.client()
        let noOrganization = try await client.getIosUserPage(request: .with { $0.id = "u3" }).result.get()
        XCTAssertTrue(noOrganization.hasUser)
        XCTAssertFalse(noOrganization.user.hasOrganization)
        XCTAssertNil(noOrganization.user.asIosUserCardFragmentData().organization)
        let missing = try await client.getIosUserPage(request: .with { $0.id = "missing" }).result.get()
        XCTAssertFalse(missing.hasUser)
        let invalid = await client.getIosUserPage(request: App_V1_GetIosUserPageRequest())
        // response.code maps HTTP status; error.code contains the decoded RPC status.
        XCTAssertEqual(invalid.error?.code, .invalidArgument)
    }
}

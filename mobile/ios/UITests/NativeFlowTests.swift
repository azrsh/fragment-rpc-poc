import XCTest

final class NativeFlowTests: XCTestCase {
    @MainActor
    func testProfileSummaryAndMissingData() {
        let app = XCUIApplication()
        app.launch()
        app.buttons["fetch"].tap()
        XCTAssertTrue(app.staticTexts["organization-name"].waitForExistence(timeout: 10))
        XCTAssertEqual(app.staticTexts["user-name"].label, "Aki Tanaka")
        XCTAssertEqual(app.staticTexts["organization-name"].label, "Northstar Studio")
        app.buttons["user-u2"].tap()
        app.segmentedControls["operation"].buttons["サマリー"].tap()
        app.buttons["fetch"].tap()
        XCTAssertTrue(app.staticTexts["user-name"].waitForExistence(timeout: 10))
        XCTAssertEqual(app.staticTexts["user-name"].label, "Ren Sato")
        XCTAssertFalse(app.staticTexts["organization-name"].exists)
        app.segmentedControls["operation"].buttons["プロフィール"].tap()
        app.buttons["user-u3"].tap()
        app.buttons["fetch"].tap()
        XCTAssertTrue(app.staticTexts["no-organization"].waitForExistence(timeout: 10))
        app.buttons["user-missing"].tap()
        app.buttons["fetch"].tap()
        XCTAssertTrue(app.staticTexts["missing-user"].waitForExistence(timeout: 10))
    }
}

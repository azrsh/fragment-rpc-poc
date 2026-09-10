package dev.fragmentrpc

import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class NativeFlowTest {
    @get:Rule val compose = createAndroidComposeRule<MainActivity>()
    private fun waitFor(text: String) {
        compose.waitUntil(10000) { compose.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty() }
    }
    @Test fun profileSummaryAndMissingData() {
        compose.onNodeWithTag("fetch").performClick()
        waitFor("Northstar Studio")
        compose.onNodeWithTag("user-name").assertTextEquals("Aki Tanaka")
        compose.onNodeWithTag("user-u2").performClick()
        compose.onNodeWithTag("summary").performClick()
        compose.onNodeWithTag("fetch").performClick()
        waitFor("Ren Sato")
        compose.onNodeWithTag("organization-name").assertDoesNotExist()
        compose.onNodeWithTag("profile").performClick()
        compose.onNodeWithTag("user-u3").performClick()
        compose.onNodeWithTag("fetch").performClick()
        waitFor("所属組織はありません。")
        compose.onNodeWithTag("user-missing").performClick()
        compose.onNodeWithTag("fetch").performClick()
        waitFor("ユーザーが見つかりません。")
    }
}

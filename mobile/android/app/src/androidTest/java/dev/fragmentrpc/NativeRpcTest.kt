package dev.fragmentrpc

import androidx.test.ext.junit.runners.AndroidJUnit4
import com.connectrpc.Code
import com.connectrpc.ConnectException
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import dev.fragmentrpc.generated.*

@RunWith(AndroidJUnit4::class)
class NativeRpcTest {
    @Test fun generatedClientCallsLiveGateway() = runBlocking {
        RpcRepository().use { api ->
            val profile = api.page("u1")
            assertTrue(profile.hasUser())
            assertEquals("Aki Tanaka", profile.user.name)
            assertEquals("Northstar Studio", profile.user.organization.name)
            assertEquals("/avatars/aki.svg", profile.user.avatarUrl)
            val card: AndroidUserCardFragmentData = profile.user.asAndroidUserCardFragmentData()
            val avatar: AndroidAvatarFragmentData = card.asAndroidAvatarFragmentData()
            assertEquals("/avatars/aki.svg", avatar.avatarUrl)
            assertEquals("Northstar Studio", card.organization?.asAndroidOrganizationBadgeFragmentData()?.name)
            assertEquals(listOf("avatarUrl"), avatar.javaClass.declaredFields.filterNot {
                it.isSynthetic || java.lang.reflect.Modifier.isStatic(it.modifiers)
            }.map { it.name })
            assertFalse(profile.toString().contains("email"))
            val summary = api.summary("u2")
            assertEquals("Ren Sato", summary.user.name)
            assertFalse(summary.toString().contains("organization"))
            assertFalse(api.page("u3").user.hasOrganization())
            assertNull(api.page("u3").user.asAndroidUserCardFragmentData().organization)
            assertFalse(api.page("missing").hasUser())
            try { api.page(null); fail("Missing id must fail") }
            catch (e: ConnectException) { assertEquals(Code.INVALID_ARGUMENT, e.code) }
        }
    }
}

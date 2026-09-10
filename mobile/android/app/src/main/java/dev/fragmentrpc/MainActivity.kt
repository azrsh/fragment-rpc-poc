package dev.fragmentrpc

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.fragmentrpc.userpage.UserPage
import dev.fragmentrpc.summary.UserSummary

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme(colorScheme = lightColorScheme(primary = Color(0xFF4961C5))) { DemoScreen() }
        }
    }
}

@Composable
fun DemoScreen(model: DemoModel = viewModel()) {
    Column(Modifier.fillMaxSize().background(Color(0xFFEDF2FA)).systemBarsPadding()
        .verticalScroll(rememberScrollState()).padding(22.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
        Text("Fragment RPC", fontSize = 19.sp, fontWeight = FontWeight.SemiBold)
        Text("フラグメントから、\nネイティブの RPC へ。", fontSize = 28.sp, fontWeight = FontWeight.SemiBold)
        Text("Compose → 専用 Protobuf API → gRPC services", fontSize = 11.sp, color = Color(0xFF64738B))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            FilterChip(selected = !model.summary, onClick = { model.summary = false }, enabled = !model.loading,
                label = { Text("プロフィール") }, modifier = Modifier.testTag("profile"))
            FilterChip(selected = model.summary, onClick = { model.summary = true }, enabled = !model.loading,
                label = { Text("サマリー") }, modifier = Modifier.testTag("summary"))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            listOf("u1", "u2", "u3", "missing").forEach { id ->
                FilterChip(selected = model.userId == id, onClick = { model.userId = id }, enabled = !model.loading,
                    label = { Text(id, fontSize = 12.sp) }, modifier = Modifier.testTag("user-$id"))
            }
        }
        Button(onClick = model::load, enabled = !model.loading, modifier = Modifier.fillMaxWidth().testTag("fetch")) {
            Text(if (model.loading) "取得中…" else "生成 API を呼び出す")
        }
        if (model.error.isNotEmpty()) Text(model.error, color = MaterialTheme.colorScheme.error, modifier = Modifier.testTag("rpc-error"))
        model.page?.let { UserPage(it) }
        model.summaryResult?.let { UserSummary(it) }
        Surface(color = Color(0xFF26344C), contentColor = Color.White, shape = MaterialTheme.shapes.medium) {
            Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(model.method.ifEmpty { "生成されたレスポンス" }, fontSize = 12.sp, fontWeight = FontWeight.Bold, modifier = Modifier.testTag("rpc-method"))
                Text(model.payload, fontFamily = FontFamily.Monospace, fontSize = 10.sp, modifier = Modifier.testTag("rpc-payload"))
            }
        }
        Text("Composable と同じファイルの宣言をビルド時に合成。\n生成済み Kotlin クライアントでバイナリー通信。", fontSize = 11.sp, color = Color(0xFF64738B))
    }
}

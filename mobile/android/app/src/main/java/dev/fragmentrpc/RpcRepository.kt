package dev.fragmentrpc

import app.v1.App
import app.v1.AppServiceClient
import com.connectrpc.ProtocolClientConfig
import com.connectrpc.extensions.GoogleJavaLiteProtobufStrategy
import com.connectrpc.impl.ProtocolClient
import com.connectrpc.okhttp.ConnectOkHttpClient
import com.connectrpc.protocols.NetworkProtocol
import kotlinx.coroutines.Dispatchers
import okhttp3.OkHttpClient
import java.util.concurrent.TimeUnit

class RpcRepository(host: String = "http://10.0.2.2:3000") : AutoCloseable {
    private val http = OkHttpClient.Builder().callTimeout(5, TimeUnit.SECONDS).build()
    private val client = AppServiceClient(ProtocolClient(
        httpClient = ConnectOkHttpClient(http),
        config = ProtocolClientConfig(
            host = host,
            serializationStrategy = GoogleJavaLiteProtobufStrategy(),
            networkProtocol = NetworkProtocol.CONNECT,
            ioCoroutineContext = Dispatchers.IO,
        ),
    ))

    suspend fun page(id: String?): App.GetAndroidUserPageResponse {
        val request = App.GetAndroidUserPageRequest.newBuilder().apply { if (id != null) setId(id) }.build()
        val response = client.getAndroidUserPage(request)
        var message: App.GetAndroidUserPageResponse? = null
        response.success { message = it.message }
        response.failure { throw it.cause }
        return checkNotNull(message)
    }

    suspend fun summary(id: String): App.GetAndroidUserSummaryResponse {
        val response = client.getAndroidUserSummary(App.GetAndroidUserSummaryRequest.newBuilder().setId(id).build())
        var message: App.GetAndroidUserSummaryResponse? = null
        response.success { message = it.message }
        response.failure { throw it.cause }
        return checkNotNull(message)
    }

    override fun close() {
        http.dispatcher.executorService.shutdown()
        http.connectionPool.evictAll()
    }
}

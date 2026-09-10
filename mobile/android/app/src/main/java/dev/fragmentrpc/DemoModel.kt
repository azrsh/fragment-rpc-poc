package dev.fragmentrpc

import app.v1.App
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.CancellationException

class DemoModel : ViewModel() {
    var userId by mutableStateOf("u1")
    var summary by mutableStateOf(false)
    var loading by mutableStateOf(false)
        private set
    var page by mutableStateOf<App.GetAndroidUserPageResponse?>(null)
        private set
    var summaryResult by mutableStateOf<App.GetAndroidUserSummaryResponse?>(null)
        private set
    var payload by mutableStateOf("API を呼び出してください。")
        private set
    var error by mutableStateOf("")
        private set
    var method by mutableStateOf("")
        private set
    private val repository = RpcRepository()

    fun load() {
        if (loading) return
        loading = true
        page = null
        summaryResult = null
        error = ""
        val id = userId
        val useSummary = summary
        viewModelScope.launch {
            try {
                if (useSummary) {
                    method = "GetAndroidUserSummary"
                    summaryResult = repository.summary(id)
                    payload = summaryResult.toString()
                } else {
                    method = "GetAndroidUserPage"
                    page = repository.page(id)
                    payload = page.toString()
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = e.message ?: e.toString()
                payload = ""
            } finally { loading = false }
        }
    }

    override fun onCleared() { repository.close() }
}

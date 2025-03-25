package ui

import androidx.compose.material.AlertDialog
import androidx.compose.material.Button
import androidx.compose.material.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

@Composable
fun GlobalAlert() {
    val alertState by AlertManager.alertState.collectAsState()

    alertState?.let { alert ->
        AlertDialog(
            onDismissRequest = { AlertManager.dismiss() },
            title = { Text(alert.title) },
            text = { Text(alert.message) },
            confirmButton = {
                Button(onClick = {
                    alert.onConfirm?.invoke() // Execute the action if provided
                    AlertManager.dismiss()   // Close the dialog
                }) {
                    Text("OK")
                }
            }
        )
    }
}
object AlertManager {
    private val _alertState = MutableStateFlow<AlertData?>(null)
    val alertState = _alertState.asStateFlow()

    fun showAlert(title: String, message: String, onConfirm: (() -> Unit)? = null) {
        _alertState.value = AlertData(title, message, onConfirm)
    }

    fun dismiss() {
        _alertState.value = null
    }
}

data class AlertData(
    val title: String,
    val message: String,
    val onConfirm: (() -> Unit)? = null
)
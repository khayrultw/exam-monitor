import androidx.compose.material.MaterialTheme
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import client.StudentClient
import ui.AlertManager
import ui.GlobalAlert
import ui.StudentApp

// Main Application
fun main() = application {
    val windowState = rememberWindowState()

    Window(
        onCloseRequest = {
            if(StudentClient.isRunning.value) {
                AlertManager.showAlert(
                    "You should not quit the app",
                    "You cannot quit the app when the exam is on going"
                )
            } else {
                exitApplication()
            }
        },
        title = "Exam Monitor - Student",
        state = windowState
    ) {
        MaterialTheme {
            GlobalAlert()
            StudentApp()
        }
    }
}

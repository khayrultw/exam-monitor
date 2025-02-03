import androidx.compose.material.MaterialTheme
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import client.StudentClient
import ui.StudentApp

// Main Application
fun main() = application {
    val windowState = rememberWindowState()

    Window(
        onCloseRequest = {
            if(StudentClient.isRunning.value) {
                println("don't")
                exitApplication()
            } else {
                exitApplication()
            }
        },
        title = "Exam Monitor - Student",
        state = windowState
    ) {
        MaterialTheme {
            StudentApp()
        }
    }
}

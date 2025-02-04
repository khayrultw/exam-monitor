import androidx.compose.material.MaterialTheme
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import server.Server
import ui.TeacherApp

fun main() = application {
    val windowState = rememberWindowState()
    val server = Server()

    Window(
        onCloseRequest = {
            server.stop()
            exitApplication()
        },
        title = "Exam Monitor - Teacher",
        state = windowState
    ) {
        MaterialTheme {
            TeacherApp(server)
        }
    }
}

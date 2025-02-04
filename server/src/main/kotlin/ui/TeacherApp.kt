package ui

import androidx.compose.foundation.layout.*
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import server.Server

@Composable
fun TeacherApp(server: Server) {
    val students = server.students

    GlobalAlert()
    Column(
        modifier = Modifier.fillMaxSize().padding(horizontal = 4.dp, vertical = 16.dp)
    ) {
        if(server.isRunning.value) {
            Row(modifier = Modifier.padding(horizontal = 12.dp)) {
                Text(
                    "Connected Students: ${students.size}",
                    style = MaterialTheme.typography.h6,
                    modifier = Modifier.padding(bottom = 4.dp)
                )
                Spacer(Modifier.weight(1f))
                Button(
                    colors = ButtonDefaults.buttonColors(backgroundColor = Color.Red),
                    onClick = {
                        AlertManager.showAlert(
                            "STOP",
                            "Are you sure you want to stop the server",
                            onConfirm = {
                                server.stop()
                            }
                        )
                    }
                ) {
                    Text("Stop", color = Color.White)
                }
            }
            StudentsDisplay(students)
        } else {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Button(onClick = { server.start() }) {
                    Text("Start")
                }
            }
        }
    }
}

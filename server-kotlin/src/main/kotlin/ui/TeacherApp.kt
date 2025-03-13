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
    var room by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    fun onClickStart(room: String) {
        try {
            server.start(room.trim().toInt())
        } catch (e: Exception) {
            errorMessage = "Failed to connect: ${e.message}"
        }
    }

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
            Column(
                Modifier.fillMaxSize().padding(16.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                if (errorMessage != null) {
                    Card(
                        backgroundColor = MaterialTheme.colors.error.copy(alpha = 0.1f),
                        modifier = Modifier.padding(16.dp)
                    ) {
                        Text(
                            errorMessage!!,
                            color = MaterialTheme.colors.error,
                            modifier = Modifier.padding(16.dp)
                        )
                    }
                    Spacer(Modifier.height(16.dp))
                }
                TextField(
                    value = room,
                    onValueChange = { room = it },
                    label = { Text("Enter room number") },
                    modifier = Modifier.widthIn(200.dp, 700.dp).fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(16.dp))
                Button(onClick = { onClickStart(room) }) {
                    Text("Start")
                }
            }
        }
    }
}

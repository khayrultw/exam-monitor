package ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import client.StudentClient
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow

// Constants
const val SERVICE_PORT = 12345
const val SCREEN_UPDATE_INTERVAL = 1000L // 1 seconds

// Student UI
@Composable
fun StudentApp() {
    val isRunning = StudentClient.isRunning.value
    val isConnected = StudentClient.isConnected.value
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var studentName by remember { mutableStateOf("") }
    var roomNumber by remember { mutableStateOf("") }


    fun onClickShare() {
        try {
            StudentClient.start(studentName, roomNumber.trim().toInt())
            errorMessage = null
        } catch (e: Exception) {
            errorMessage = "Failed to connect: ${e.message}"
        }
    }
    Column(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Error message display
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

        // Connection button
        if (!isRunning) {
            Column(modifier = Modifier.padding(16.dp)) {
                TextField(
                    value = studentName,
                    onValueChange = { studentName = it },
                    label = { Text("Enter your name") },
                    modifier = Modifier.widthIn(200.dp, 700.dp).fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(16.dp))

                TextField(
                    value = roomNumber,
                    onValueChange = { roomNumber = it },
                    label = { Text("Enter room number") },
                    modifier = Modifier.widthIn(200.dp, 700.dp).fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(16.dp))

                Button(
                    onClick = { onClickShare() },
                    modifier = Modifier.width(200.dp)
                ) {
                    Text("Connect to Teacher")
                }
            }
        } else {
            if(!isConnected) {
                Box(
                    modifier = Modifier.background(MaterialTheme.colors.error.copy(alpha = 0.1f)),
                ) {
                    Text(
                        "Looking for the host...",
                        modifier = Modifier.padding(16.dp)
                    )
                }
                Spacer(Modifier.height(16.dp))
                Button(
                    onClick = {
                        StudentClient.stop()
                    },
                    colors = ButtonDefaults.buttonColors(
                        backgroundColor = MaterialTheme.colors.error
                    ),
                    modifier = Modifier.width(200.dp)
                ) {
                    Text("Stop")
                }
                Spacer(Modifier.height(16.dp))
            } else {
                Button(
                    onClick = {
                        StudentClient.stop()
                    },
                    colors = ButtonDefaults.buttonColors(
                        backgroundColor = MaterialTheme.colors.error
                    ),
                    modifier = Modifier.width(200.dp)
                ) {
                    Text("Stop Sharing")
                }

                Spacer(Modifier.height(16.dp))
                Text(
                    "Screen is being shared",
                    color = MaterialTheme.colors.primary
                )
            }
        }
    }
}

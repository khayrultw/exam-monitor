package ui

import androidx.compose.foundation.layout.*
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import core.Constants
import data.Student
import kotlinx.coroutines.*
import server.Server

@Composable
fun TeacherApp(server: Server) {
    var students by remember { mutableStateOf<List<Student>>(emptyList()) }
    LaunchedEffect(Unit) {
        while (true) {
            students = server.getStudentScreens()
            delay(Constants.SCREEN_UPDATE_INTERVAL)
        }
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(horizontal = 4.dp, vertical = 16.dp)
    ) {
        Text(
            "Connected Students: ${students.size}",
            style = MaterialTheme.typography.h6,
            modifier = Modifier.padding(bottom = 4.dp)
        )
        StudentsDisplay(students)
    }
}

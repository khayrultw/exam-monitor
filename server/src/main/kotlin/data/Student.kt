package data

import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import java.awt.image.BufferedImage
import java.net.Socket

data class Student(
    val id: String,
    val name: MutableState<String> = mutableStateOf(""),
    val message: MutableState<String> = mutableStateOf(""),
    val socket: Socket,
    val lastImage: MutableState<BufferedImage?> = mutableStateOf(null)
)
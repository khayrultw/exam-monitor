package server

import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.snapshots.SnapshotStateList
import core.Constants
import core.Constants.SCREEN_UPDATE_INTERVAL
import data.Student
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.ByteArrayInputStream
import java.io.DataInputStream
import java.io.File
import java.io.OutputStream
import java.net.ServerSocket
import java.net.StandardSocketOptions
import javax.imageio.ImageIO

class Server {
    val students: SnapshotStateList<Student> = mutableStateListOf()
    private var serverSocket: ServerSocket? = null
    val isRunning = mutableStateOf(false)
    private val scope = CoroutineScope(Dispatchers.IO)

    fun start() {
        serverSocket = ServerSocket(Constants.SERVICE_PORT)
        isRunning.value = true
        scope.launch {
            while (isRunning.value) {
                try {
                    val socket = serverSocket?.accept() ?: continue
                    socket.keepAlive = true
                    socket.soTimeout = 5000
                    socket.setOption(StandardSocketOptions.TCP_NODELAY, true)
                    socket.setOption(StandardSocketOptions.SO_KEEPALIVE, true)
                    socket.setOption(StandardSocketOptions.SO_REUSEADDR, true)
                    val student = Student(
                        id = "Student${students.size + 1}",
                        socket = socket
                    )
                    students.add(student)
                    handleStudent(student)
                    delay(SCREEN_UPDATE_INTERVAL)
                }
                catch (e: Exception) {
                    if (isRunning.value) e.printStackTrace()
                }
            }
        }
    }

    private fun handleStudent(student: Student) {
        scope.launch {
            val input = DataInputStream(student.socket.getInputStream())
            var stream: OutputStream? = null
            while (isRunning.value) {
                try {
                    val type = input.readInt()
                    when(type) {
                        0 -> {
                            val size = input.readInt()
                            val bytes = ByteArray(size)
                            input.readFully(bytes)
                            val name = String(bytes, Charsets.UTF_8)
                            stream = createVideoStream(name)
                            student.name.value = name
                        }

                        1 -> {
                            val size = input.readInt()
                            val bytes = ByteArray(size)
                            input.readFully(bytes)
                            val msg = String(bytes, Charsets.UTF_8)
                            student.message.value = msg
                        }
                        else -> {
                            val size = input.readInt()
                            val bytes = ByteArray(size)
                            input.readFully(bytes)
                            val image = ImageIO.read(ByteArrayInputStream(bytes))
                            student.lastImage.value = image
                            if(stream != null) {
                                scope.launch {
                                    try {
                                        ImageIO.write(image, "jpeg", stream)
                                    } catch (e: Exception) {
                                        e.printStackTrace()
                                    }
                                }
                            }
                        }
                    }
                } catch (e: Exception) {
                    break
                }
            }
            student.socket.close()
            stream?.close()
            students.remove(student)
        }
    }

    private fun createVideoStream(name: String): OutputStream {
        val sanitizedName = name.replace(Regex("[^a-zA-Z0-9_-]"), "_")

        val folder = File("student_images/$sanitizedName")
        if (!folder.exists()) folder.mkdirs()

        val outputFile = File(folder, "${System.currentTimeMillis()}.mp4").absolutePath

        val process = ProcessBuilder(
            "ffmpeg",
            "-y", "-f", "image2pipe", "-vcodec", "mjpeg",
            "-r", "4", "-i", "-",
            "-vcodec", "libx264", "-pix_fmt", "yuv420p",
            "-preset", "veryfast", "-b:v", "500k",
            outputFile
        ).redirectErrorStream(true).start()

        return process.outputStream
    }


    fun stop() {
        isRunning.value = false
        students.forEach { it.socket.close() }
        serverSocket?.close()
    }
}

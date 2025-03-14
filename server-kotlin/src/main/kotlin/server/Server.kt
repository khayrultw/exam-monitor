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
import java.awt.image.BufferedImage
import java.io.ByteArrayInputStream
import java.io.DataInputStream
import java.io.File
import java.io.IOException
import java.io.OutputStream
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.ServerSocket
import java.net.StandardSocketOptions
import javax.imageio.ImageIO

class Server {
    val students: SnapshotStateList<Student> = mutableStateListOf()
    private var serverSocket: ServerSocket? = null
    val isRunning = mutableStateOf(false)
    private val scope = CoroutineScope(Dispatchers.IO)

    fun start(port: Int) {
        broadcastHost(port)
        serverSocket = ServerSocket(port)
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

    private fun broadcastHost(port: Int) {
        scope.launch {
            val socket = DatagramSocket()
            val address = InetAddress.getByName("255.255.255.255")  // Broadcast address
            val message = "server"
            while (isRunning.value) {
                val packet = DatagramPacket(
                    message.toByteArray(),
                    message.length,
                    address,
                    port
                )
                try {
                    socket.send(packet)
                } catch (_: Exception) {}
                Thread.sleep(2000)  // Send every 2 seconds
            }
        }
    }

    private fun handleStudent(student: Student) {
        scope.launch {
            val input = DataInputStream(student.socket.getInputStream())
            //var stream: OutputStream? = null
            while (isRunning.value) {
                try {
                    val type = input.readInt()
                    when(type) {
                        0 -> {
                            val size = input.readInt()
                            val bytes = ByteArray(size)
                            input.readFully(bytes)
                            val name = String(bytes, Charsets.UTF_8)
                            // stream = createVideoStream(name)
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
                            saveImage(student.name.value, image)
                        }
                    }
                } catch (e: Exception) {
                    break
                }
            }
            student.socket.close()
            students.remove(student)
        }
    }

    private fun saveImage(name: String, image: BufferedImage) {
        val sanitizedName = name.replace(Regex("[^a-zA-Z0-9_-]"), "_")

        val folder = File("student_images/$sanitizedName")
        if (!folder.exists()) folder.mkdirs()

        val timestamp = System.currentTimeMillis()
        val file = File(folder, "$timestamp.png")

        try {
            ImageIO.write(image, "png", file)
        } catch (e: Exception) {
            e.printStackTrace()
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

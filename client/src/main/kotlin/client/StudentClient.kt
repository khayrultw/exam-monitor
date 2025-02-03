package client

import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import ui.SCREEN_UPDATE_INTERVAL
import ui.SERVICE_PORT
import java.awt.Rectangle
import java.awt.Robot
import java.awt.Toolkit
import java.awt.image.BufferedImage
import java.io.ByteArrayOutputStream
import java.io.DataOutputStream
import java.net.Socket
import javax.imageio.ImageIO

object StudentClient {
    var isRunning: MutableState<Boolean> = mutableStateOf(false)
    private var dataStream: DataOutputStream? = null
    private var socket: Socket? = null
    private val robot = Robot()
    private var job: Job? = null

    fun start(serverAddress: String, studentName: String) {
        socket = Socket(serverAddress, SERVICE_PORT)
        isRunning.value = true
        dataStream = socket?.let { DataOutputStream(it.getOutputStream()) }
        val scope = CoroutineScope(Dispatchers.IO)
        job = scope.launch {
            sendStudentName(studentName)
            while (isRunning.value) {
                try {
                    val screenshot = captureScreen()
                    sendScreenshot(screenshot)
                    delay(SCREEN_UPDATE_INTERVAL)
                } catch (e: Exception) {
                    if (isRunning.value) e.printStackTrace()
                    isRunning.value = false
                    socket?.close()
                    break
                }
            }
        }
    }

    private fun captureScreen(): BufferedImage {
        val screenRect = Rectangle(Toolkit.getDefaultToolkit().screenSize) // Adjust to actual screen size
        return robot.createScreenCapture(screenRect)
    }

    private fun sendStudentName(name: String) {
        dataStream?.let { stream ->
            val nameBytes = name.toByteArray(Charsets.UTF_8)
            stream.writeInt(0)
            stream.writeInt(nameBytes.size)
            stream.write(nameBytes)
            stream.flush()
        }
    }

    fun sendMessage(msg: String) {
        dataStream?.let { stream ->
            val msgBytes = msg.toByteArray(Charsets.UTF_8)
            stream.writeInt(1)
            stream.writeInt(msgBytes.size)
            stream.write(msgBytes)
            stream.flush()
        }
    }

    private fun sendScreenshot(image: BufferedImage) {
        dataStream?.let { stream ->
            val baos = ByteArrayOutputStream()
            ImageIO.write(image, "jpg", baos)
            val bytes = baos.toByteArray()
            stream.writeInt(2)
            stream.writeInt(bytes.size)
            stream.write(bytes)
            stream.flush()
        }
    }

    fun stop() {
        isRunning.value = false
        socket?.close()
        job?.cancel()
    }
}

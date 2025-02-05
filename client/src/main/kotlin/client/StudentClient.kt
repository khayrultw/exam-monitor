package client

import androidx.compose.runtime.MutableState
import androidx.compose.runtime.internal.isLiveLiteralsEnabled
import androidx.compose.runtime.mutableStateOf
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import ui.SCREEN_UPDATE_INTERVAL
import ui.SERVICE_PORT
import java.awt.Rectangle
import java.awt.Robot
import java.awt.Toolkit
import java.awt.image.BufferedImage
import java.io.ByteArrayOutputStream
import java.io.DataOutputStream
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.Socket
import java.net.SocketException
import javax.imageio.ImageIO
import javax.imageio.ImageWriteParam

object StudentClient {
    var isRunning: MutableState<Boolean> = mutableStateOf(false)
    var isConnected: MutableState<Boolean> = mutableStateOf(false)
    private var dataStream: DataOutputStream? = null
    private var socket: Socket? = null
    private val imageWriter = ImageIO.getImageWritersByFormatName("jpg").next()
    private val robot = Robot()
    private val scope = CoroutineScope(Dispatchers.IO)
    private var job: Job? = null

    fun start(studentName: String, port: Int) {
        isRunning.value = true
        job = scope.launch {
            var retryDelay = 1000L
            while (isRunning.value) {  // Keep trying to connect
                try {
                    val serverAddress = findServer(port) // Keep searching until found
                    isConnected.value = true
                    retryDelay = 1000L
                    socket = Socket(serverAddress, port)
                    dataStream = socket?.getOutputStream()?.let { DataOutputStream(it) }

                    sendStudentName(studentName)

                    while (isRunning.value) {
                        val screenshot = captureScreen()
                        sendScreenshot(screenshot)
                        delay(SCREEN_UPDATE_INTERVAL) // Wait before next screenshot
                    }

                }catch (e: Exception) {
                    e.printStackTrace()
                    isConnected.value = false
                    retryDelay = (retryDelay * 2).coerceAtMost(8000)
                    delay(retryDelay)
                } finally {
                    socket?.close() // Close socket before retrying
                }
            }
        }
    }

    private suspend fun findServer(port: Int): String = withContext(Dispatchers.IO) {
        val buffer = ByteArray(256)
        val packet = DatagramPacket(buffer, buffer.size)

        while (isRunning.value) {  // Keep searching until `isLooking` is false
            try {
                DatagramSocket(port).use { socket ->
                    socket.soTimeout = 5000  // Timeout every 5s to allow retry
                    socket.receive(packet)
                    return@withContext packet.address.hostAddress
                }
            } catch (e: Exception) {
                delay(2000)  // Wait 2 seconds before retrying
            }
        }

        throw Exception("Server search was stopped manually.")
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

            val ios = ImageIO.createImageOutputStream(baos)
            imageWriter.output = ios

            val params = imageWriter.defaultWriteParam
            params.compressionMode = ImageWriteParam.MODE_EXPLICIT
            params.compressionQuality = 0.4f  // Lower quality (0.0 = worst, 1.0 = best)

            imageWriter.write(null, javax.imageio.IIOImage(image, null, null), params)
            ios.close()

            val bytes = baos.toByteArray()
            stream.writeInt(2)
            stream.writeInt(bytes.size)
            stream.write(bytes)
            stream.flush()
        }
    }


    fun stop() {
        job?.cancel()
        isRunning.value = false
        isConnected.value = false
        socket?.close()
    }
}

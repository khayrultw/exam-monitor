package server

import core.Constants
import core.Constants.SCREEN_UPDATE_INTERVAL
import data.Student
import java.io.ByteArrayInputStream
import java.io.DataInputStream
import java.io.EOFException
import java.net.ServerSocket
import java.util.concurrent.ConcurrentHashMap
import javax.imageio.ImageIO
import kotlin.collections.set

class Server {
    private val students = ConcurrentHashMap<String, Student>()
    private var serverSocket: ServerSocket? = null
    private var isRunning = true

    fun start() {
        serverSocket = ServerSocket(Constants.SERVICE_PORT)
        println("Teacher server started on port ${Constants.SERVICE_PORT}")

        Thread {
            while (isRunning) {
                try {
                    val socket = serverSocket?.accept() ?: continue
                    val student = Student(
                        id = "Student${students.size + 1}",
                        socket = socket
                    )
                    students[student.id] = student
                    handleStudent(student)
                    Thread.sleep(SCREEN_UPDATE_INTERVAL)
                } catch (e: Exception) {
                    if (isRunning) e.printStackTrace()
                }
            }
        }.start()
    }

    private fun handleStudent(student: Student) {
        Thread {
            val input = DataInputStream(student.socket.getInputStream())

            while (isRunning) {
                try {
                    val type = input.readInt()
                    when(type) {
                        0 -> {
                            val size = input.readInt()
                            val bytes = ByteArray(size)
                            input.readFully(bytes)
                            val name = String(bytes, Charsets.UTF_8)
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
                            student.lastImage.value = ImageIO.read(ByteArrayInputStream(bytes))
                        }
                    }
                } catch (e: Exception) {
                    students.remove(student.id)
                    break
                }
            }
        }.start()
    }


    fun getStudentScreens(): List<Student> = students.values.toList()

    fun stop() {
        isRunning = false
        students.forEach { it.value.socket.close() }
        serverSocket?.close()
    }
}

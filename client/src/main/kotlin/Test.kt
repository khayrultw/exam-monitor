import java.awt.Rectangle
import java.awt.Robot
import java.awt.Toolkit
import java.awt.image.BufferedImage
import java.io.File
import javax.imageio.ImageIO

fun captureScreen() {
    try {
        val screenRect = Rectangle(Toolkit.getDefaultToolkit().screenSize)
        val capture: BufferedImage = Robot().createScreenCapture(screenRect)
        val outputFile = File("screenshot.png")
        ImageIO.write(capture, "png", outputFile)
        println("Screenshot saved to: ${outputFile.absolutePath}")
    } catch (e: Exception) {
        e.printStackTrace()
        println("Failed to capture screen: ${e.message}")
    }
}

fun main() {
    captureScreen()
}
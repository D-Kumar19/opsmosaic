import AppKit
import ImageIO
import UniformTypeIdentifiers

enum DemoError: Error {
    case invalidArguments, fontUnavailable, contextUnavailable, imageUnavailable
    case destinationUnavailable, encodingFailed, verificationFailed
}

guard CommandLine.arguments.count == 3 else { throw DemoError.invalidArguments }
let directory = URL(fileURLWithPath: CommandLine.arguments[1])
let output = URL(fileURLWithPath: CommandLine.arguments[2])
guard let font = NSFont(name: "Menlo", size: 15) else { throw DemoError.fontUnavailable }
let width = 1100
let height = 880
let titles = [
    "First run / example profile",
    "Choose interests / one random tip",
    "Return offline / no onboarding or network"
]
let durations = [3.0, 10.0, 8.0]
var images: [CGImage] = []
for index in 0..<3 {
    let text = try String(contentsOf: directory.appendingPathComponent("scene-\(index).txt"),
                          encoding: .utf8)
    guard let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: width,
        pixelsHigh: height, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true,
        isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0),
        let context = NSGraphicsContext(bitmapImageRep: bitmap)
    else { throw DemoError.contextUnavailable }
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    NSColor(calibratedRed: 0.055, green: 0.07, blue: 0.10, alpha: 1).setFill()
    NSRect(x: 0, y: 0, width: width, height: height).fill()
    let header: [NSAttributedString.Key: Any] = [
        .font: NSFont.boldSystemFont(ofSize: 20),
        .foregroundColor: NSColor(calibratedRed: 0.4, green: 0.85, blue: 0.9, alpha: 1)
    ]
    ("OpsMosaic — " + titles[index]).draw(at: NSPoint(x: 24, y: height - 48),
                                         withAttributes: header)
    let style = NSMutableParagraphStyle()
    style.minimumLineHeight = 21
    style.maximumLineHeight = 21
    let attributes: [NSAttributedString.Key: Any] = [
        .font: font, .foregroundColor: NSColor(calibratedWhite: 0.9, alpha: 1),
        .paragraphStyle: style
    ]
    let content = NSAttributedString(string: text, attributes: attributes)
    let rect = NSRect(x: 24, y: 55, width: width - 48, height: height - 125)
    let bounds = content.boundingRect(with: rect.size,
        options: [.usesLineFragmentOrigin, .usesFontLeading])
    guard bounds.height <= rect.height else { throw DemoError.verificationFailed }
    content.draw(with: rect, options: [.usesLineFragmentOrigin, .usesFontLeading])
    ("Example profile • actual CLI capture • no network • \(index + 1)/3").draw(
        at: NSPoint(x: 24, y: 20),
        withAttributes: [.font: font, .foregroundColor: NSColor.lightGray])
    NSGraphicsContext.restoreGraphicsState()
    guard let image = bitmap.cgImage else { throw DemoError.imageUnavailable }
    images.append(image)
}
guard let destination = CGImageDestinationCreateWithURL(output as CFURL,
    UTType.gif.identifier as CFString, images.count, nil) else {
    throw DemoError.destinationUnavailable
}
CGImageDestinationSetProperties(destination,
    [kCGImagePropertyGIFDictionary: [kCGImagePropertyGIFLoopCount: 0]] as CFDictionary)
for (index, image) in images.enumerated() {
    CGImageDestinationAddImage(destination, image,
        [kCGImagePropertyGIFDictionary: [kCGImagePropertyGIFDelayTime: durations[index]]] as CFDictionary)
}
guard CGImageDestinationFinalize(destination) else { throw DemoError.encodingFailed }
guard let source = CGImageSourceCreateWithURL(output as CFURL, nil),
    CGImageSourceGetCount(source) == 3 else { throw DemoError.verificationFailed }
for index in 0..<3 {
    guard let image = CGImageSourceCreateImageAtIndex(source, index, nil),
        image.width == width, image.height == height else { throw DemoError.verificationFailed }
    guard let properties = CGImageSourceCopyPropertiesAtIndex(source, index, nil) as? [CFString: Any],
        let gif = properties[kCGImagePropertyGIFDictionary] as? [CFString: Any],
        let delay = gif[kCGImagePropertyGIFDelayTime] as? Double,
        abs(delay - durations[index]) < 0.01 else { throw DemoError.verificationFailed }
}
print("GIF verified: 3 frames, \(width)x\(height), 21-second loop.")

$chrome = "C:\Program Files\Google\Chrome\Application\chrome.exe"
$framesDir = "B:\workgit\cassettecraft\temp_frames"
$docsImages = "B:\workgit\cassettecraft\docs\images"

if (!(Test-Path $framesDir)) {
    New-Item -ItemType Directory -Path $framesDir -Force
}

# 1. Waterfall
Start-Process -FilePath $chrome -ArgumentList "--headless=new", "--screenshot=$framesDir\f1.png", "--window-size=1280,800", "http://localhost:8585/#waterfall" -Wait -NoNewWindow
Start-Sleep -Milliseconds 500

# 2. Cassettes
Start-Process -FilePath $chrome -ArgumentList "--headless=new", "--screenshot=$framesDir\f2.png", "--window-size=1280,800", "http://localhost:8585/#cassettes" -Wait -NoNewWindow
Start-Sleep -Milliseconds 500

# 3. Scratchpad
Start-Process -FilePath $chrome -ArgumentList "--headless=new", "--screenshot=$framesDir\f3.png", "--window-size=1280,800", "http://localhost:8585/#scratchpad" -Wait -NoNewWindow
Start-Sleep -Milliseconds 500

# 4. Drift
Start-Process -FilePath $chrome -ArgumentList "--headless=new", "--screenshot=$framesDir\f4.png", "--window-size=1280,800", "http://localhost:8585/#drift" -Wait -NoNewWindow
Start-Sleep -Milliseconds 500

# 5. Switch to Replay mode
curl.exe -s -X POST http://localhost:8585/api/v1/mode -H "Content-Type: application/json" -d '{\"mode\":\"replay\"}'
Start-Sleep -Milliseconds 500

# 6. Waterfall in Replay mode
Start-Process -FilePath $chrome -ArgumentList "--headless=new", "--screenshot=$framesDir\f5.png", "--window-size=1280,800", "http://localhost:8585/#waterfall" -Wait -NoNewWindow
Start-Sleep -Milliseconds 500

# Switch back to auto
curl.exe -s -X POST http://localhost:8585/api/v1/mode -H "Content-Type: application/json" -d '{\"mode\":\"auto\"}'

# Generate sequence for 2.0s per slide
$sequence = @("f1.png", "f1.png", "f1.png", "f1.png",
              "f2.png", "f2.png", "f2.png", "f2.png",
              "f3.png", "f3.png", "f3.png", "f3.png",
              "f4.png", "f4.png", "f4.png", "f4.png",
              "f5.png", "f5.png", "f5.png", "f5.png")

$idx = 0
foreach ($f in $sequence) {
    $src = Join-Path $framesDir $f
    $dstName = "seq_{0:D3}.png" -f $idx
    $dst = Join-Path $framesDir $dstName
    Copy-Item -Path $src -Destination $dst -Force
    $idx++
}

# Run ffmpeg
$palette = "$framesDir\palette.png"
$gifOut = "$docsImages\cassettecraft_demo.gif"

ffmpeg.exe -y -framerate 2 -i "$framesDir\seq_%03d.png" -vf "fps=2,scale=1000:-1:flags=lanczos,palettegen" $palette
ffmpeg.exe -y -framerate 2 -i "$framesDir\seq_%03d.png" -i $palette -lavfi "fps=2,scale=1000:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=5" $gifOut

Write-Host "✓ Generated cassettecraft_demo.gif successfully."

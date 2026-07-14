/**
 * Squoosh 圖片壓縮對比工具
 * 使用 Canvas API + Layui slider 實現即時 WebP 壓縮 + 前後對比
 * 配置從 gbootImageConfig 全域變數讀取（由 foot.html 注入）
 */
layui.define(['layer', 'slider'], function(exports) {
    var layer = layui.layer;
    var slider = layui.slider;
    var $ = layui.$;

    var Squoosh = {
        /**
         * 打開壓縮對比框
         * @param {File} file 原始圖片文件
         * @param {Function} callback 壓縮完成回調，參數為壓縮後的 File（或原始 File 表示跳過）
         */
        open: function(file, callback) {
            // 從全域配置讀取預設值
            var cfg = (typeof gbootImageConfig !== 'undefined') ? gbootImageConfig : {};
            var quality = parseInt(cfg.quality) || 80;
            var maxWidth = parseInt(cfg.maxWidth) || 1920;

            var originalBlob = file;
            var originalURL = null;
            var compressedBlob = null;
            var compressedURL = null;
            var img = null;
            var layerIndex = null;
            var sliderInst = null;

            // 瀏覽器 WebP 編碼能力檢測（只檢測一次）
            var webpSupported = (function() {
                var c = document.createElement('canvas');
                c.width = 1; c.height = 1;
                return c.toDataURL('image/webp').indexOf('data:image/webp') === 0;
            })();

            initModal();

            function initModal() {
                originalURL = URL.createObjectURL(originalBlob);
                img = new Image();
                img.onload = function() {
                    renderModal();
                    compressPreview();
                };
                img.src = originalURL;
            }

            function renderModal() {
                var origSize = formatSize(originalBlob.size);
                var w = img.naturalWidth;
                var h = img.naturalHeight;

                var html = [
                    '<div class="squoosh-container" style="padding:16px;">',
                    // 對比區域
                    '<div class="squoosh-compare" style="position:relative;width:100%;max-height:400px;overflow:hidden;border:1px solid #e6e6e6;border-radius:4px;background:#f2f2f2;user-select:none;">',
                    '<div class="squoosh-after" style="width:100%;height:400px;display:flex;align-items:center;justify-content:center;">',
                    '<img id="squooshImgAfter" style="max-width:100%;max-height:400px;" />',
                    '</div>',
                    '<div class="squoosh-before" style="position:absolute;top:0;left:0;width:50%;height:100%;overflow:hidden;border-right:2px solid #1E9FFF;">',
                    '<img id="squooshImgBefore" style="max-width:1920px;max-height:400px;" />',
                    '</div>',
                    '<div class="squoosh-divider" style="position:absolute;top:0;left:50%;width:2px;height:100%;background:#1E9FFF;cursor:ew-resize;z-index:10;">',
                    '<div style="position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);width:32px;height:32px;background:#1E9FFF;border-radius:50%;display:flex;align-items:center;justify-content:center;color:#fff;font-size:14px;">\u21d4</div>',
                    '</div>',
                    '<div style="position:absolute;top:8px;left:8px;background:rgba(0,0,0,0.6);color:#fff;padding:2px 8px;border-radius:3px;font-size:12px;">原始</div>',
                    '<div style="position:absolute;top:8px;right:8px;background:rgba(30,159,255,0.8);color:#fff;padding:2px 8px;border-radius:3px;font-size:12px;">壓縮後</div>',
                    '</div>',
                    // 控制區域
                    '<div style="margin-top:12px;">',
                    '<div class="layui-form-item" style="margin-bottom:0;">',
                    '<label class="layui-form-label" style="width:80px;">質量</label>',
                    '<div class="layui-input-inline" style="width:300px;"><div id="squooshSlider"></div></div>',
                    '<div class="layui-input-inline" style="width:60px;"><input type="text" id="squooshQuality" class="layui-input" style="text-align:center;" value="' + quality + '" /></div>',
                    '<div class="layui-form-mid layui-word-aux">1-100，Google 建議 75-80</div>',
                    '</div>',
                    '</div>',
                    // 信息區域
                    '<div style="margin-top:12px;display:flex;justify-content:space-between;align-items:center;">',
                    '<div>',
                    '<span style="color:#666;">原始：</span><span id="squooshOrigInfo">' + w + '\u00d7' + h + ' / ' + origSize + '</span>',
                    '<span style="margin-left:16px;color:#666;">壓縮後：</span><span id="squooshCompInfo" style="color:#1E9FFF;">計算中...</span>',
                    '<span style="margin-left:16px;color:#16b777;" id="squooshSaved"></span>',
                    '</div>',
                    '<div>',
                    '<span style="color:#999;font-size:12px;">拖動中間分隔線對比效果</span>',
                    '</div>',
                    '</div>',
                    '</div>'
                ].join('');

                layerIndex = layer.open({
                    type: 1,
                    title: '圖片壓縮對比',
                    area: ['720px', '600px'],
                    shade: 0.3,
                    content: html,
                    btn: ['使用壓縮版本', '使用原圖'],
                    btn1: function() {
                        if (compressedBlob) {
                            var newName = originalBlob.name.replace(/\.(jpg|jpeg|png)$/i, '.webp');
                            var compressedFile = new File([compressedBlob], newName, { type: 'image/webp' });
                            cleanup();
                            callback(compressedFile);
                        } else {
                            cleanup();
                            callback(originalBlob);
                        }
                    },
                    btn2: function() {
                        cleanup();
                        callback(originalBlob);
                    },
                    cancel: function() {
                        cleanup();
                        callback(originalBlob);
                    }
                });

                // 設置原始圖片
                $('#squooshImgBefore').attr('src', originalURL);
                $('#squooshImgAfter').attr('src', originalURL);

                // 渲染 Layui slider
                sliderInst = slider.render({
                    elem: '#squooshSlider',
                    value: quality,
                    min: 1,
                    max: 100,
                    step: 1,
                    theme: '#1E9FFF',
                    change: function(val) {
                        $('#squooshQuality').val(val);
                        quality = val;
                        compressPreview();
                    }
                });

                $('#squooshQuality').on('input', function() {
                    var v = parseInt($(this).val());
                    if (v >= 1 && v <= 100) {
                        quality = v;
                        sliderInst.setValue(v);
                        compressPreview();
                    }
                });

                // 對比滑動
                var dragging = false;
                $('.squoosh-divider').on('mousedown', function(e) {
                    dragging = true;
                    e.preventDefault();
                });
                $(document).on('mousemove.squoosh', function(e) {
                    if (!dragging) return;
                    var container = $('.squoosh-compare');
                    var offset = container.offset();
                    var x = e.pageX - offset.left;
                    var pct = (x / container.width()) * 100;
                    pct = Math.max(0, Math.min(100, pct));
                    $('.squoosh-before').css('width', pct + '%');
                    $('.squoosh-divider').css('left', pct + '%');
                });
                $(document).on('mouseup.squoosh', function() {
                    dragging = false;
                });
            }

            // 壓縮預覽（防抖 200ms）
            var compressTimer = null;
            function compressPreview() {
                clearTimeout(compressTimer);
                compressTimer = setTimeout(doCompress, 200);
            }

            function doCompress() {
                var canvas = document.createElement('canvas');
                var ctx = canvas.getContext('2d');

                var w = img.naturalWidth;
                var h = img.naturalHeight;

                // 等比縮放
                if (maxWidth > 0 && w > maxWidth) {
                    h = Math.round(h * maxWidth / w);
                    w = maxWidth;
                }

                canvas.width = w;
                canvas.height = h;

                // 白色背景（處理 PNG 透明通道）
                ctx.fillStyle = '#ffffff';
                ctx.fillRect(0, 0, w, h);
                ctx.drawImage(img, 0, 0, w, h);

                // 優先 WebP，不支援則降級 JPEG
                var mimeType = webpSupported ? 'image/webp' : 'image/jpeg';

                canvas.toBlob(function(blob) {
                    if (!blob) {
                        $('#squooshCompInfo').text('壓縮失敗');
                        return;
                    }
                    compressedBlob = blob;
                    if (compressedURL) URL.revokeObjectURL(compressedURL);
                    compressedURL = URL.createObjectURL(blob);
                    $('#squooshImgAfter').attr('src', compressedURL);

                    var compSize = formatSize(blob.size);
                    var saved = 0;
                    if (originalBlob.size > 0) {
                        saved = Math.round((1 - blob.size / originalBlob.size) * 100);
                    }
                    $('#squooshCompInfo').text(w + '\u00d7' + h + ' / ' + compSize);
                    if (saved > 0) {
                        $('#squooshSaved').text('節省 ' + saved + '%');
                    } else {
                        $('#squooshSaved').text('已是最優');
                    }
                }, mimeType, quality / 100);
            }

            function cleanup() {
                if (originalURL) URL.revokeObjectURL(originalURL);
                if (compressedURL) URL.revokeObjectURL(compressedURL);
                $(document).off('mousemove.squoosh mouseup.squoosh');
            }

            function formatSize(bytes) {
                if (bytes < 1024) return bytes + ' B';
                if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
                return (bytes / 1048576).toFixed(1) + ' MB';
            }
        }
    };

    exports('squoosh', Squoosh);
});

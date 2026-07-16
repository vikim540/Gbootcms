layui.use(['element','upload','laydate','form'], function(){
  var element = layui.element;
  var upload = layui.upload;
  var laydate = layui.laydate;
  var form = layui.form;
  
  //獲取hash來切換選項卡，假設當前地址的hash為lay-id對應的值
  var layid = location.hash.replace(/^#tab=/, '');
  element.tabChange('tab', layid); //假設當前地址為：http://a.com#test1=222，那麼選項卡會自動切換到"發送訊息"這一項
  
  //監聽Tab切換，以改變地址hash值
  element.on('tab(tab)', function(){
	var clayid=this.getAttribute('lay-id');
	if(clayid){
		location.hash = 'tab='+ clayid;
		$('.page').find('a').each(function(index,element){//避免tab翻頁問題
			var url=$(this).attr('href');
			if(url.indexOf('tab=')==-1){
				$(this).attr('href', url+'#tab='+ clayid);
			}else{
				$(this).attr('href', url.replace(/tab=[\w]+/, 'tab='+ clayid));
			}
        });
	}
  });
  
  //跳轉
	form.on('select(tourl)', function(data){
		window.location.href= data.value;
	}); 

  
  //提示
  $(".tips").on("mouseover",function(){
	layer.tips($(this).data('content'), this);
  }) 
 
  //用戶登入驗證
  form.on('submit(login-submit)', function(data){
  	var form = $("#dologin");
    var url = form.attr('action');
    var username = form.find("#username").val();
    var password = form.find("#password").val();
    var checkcode = form.find("#checkcode").val();
    var formcheck = form.find("#formcheck").val();
    
	$.ajax({
	  type: 'POST',
	  url: url,
	  dataType: 'json',
	  data: {
            username: username,
            password: password,
            checkcode: checkcode,
            formcheck: formcheck
       },
	  success: function (response, status) {
			if (response.code == 1) {
				layer.msg("登錄成功！", {icon: 1});
				window.location.href = response.data;
			} else {
				if (form.find("#checkcode").length) { form.find("#checkcode").val(""); }
				if ($('#codeimg').length) { $('#codeimg').click(); }//更新驗證碼
				layer.msg("登錄失敗：" + (response.msg || '未知錯誤'), {icon: 2});
			} 
      },
      error:function(xhr,status,error){
    	  layer.msg("登錄請求發生錯誤！", {icon: 5});
    	  $('#note').html('登錄請求失敗，請檢查網絡連接或稍後重試。');
      }
	});
    return false;
  });
  
  // ─── 通用頂級通知（設計：玻璃質感 + 左側色條 + 縮放入場） ───
  //  呼叫時只傳純文字，不再傳 icon（showNotify 內部統一處理 icon）
  function showNotify(text, type) {
      var isErr = type === 'error';
      // error → fa-times-circle · success → fa-check-circle
      var icon   = isErr ? 'fa-times-circle' : 'fa-check-circle';
      var accent = isErr ? '#ff4d4f' : '#52c41a';
      var bg     = isErr ? '#fff2f0' : '#f6ffed';
      var txtCol = isErr ? '#cf1322' : '#389e0d';
      layer.open({
          type: 1, title: false, closeBtn: 0, shade: 0,
          area: 'auto', offset: '28px',
          anim: -1, // 關閉默認動畫，用 CSS 自定義
          time: isErr ? 4000 : 2500,
          success: function(lyr) {
              // 在 .layui-layer 容器直接套圓角+溢出裁切，隱藏 LayUI 預設白背景
              $(lyr).css({ overflow: 'hidden', borderRadius: '12px' });
              // 自定義縮放入場
              $(lyr).css({ opacity: 0, transform: 'scale(0.88) translateY(-10px)' });
              setTimeout(function() {
                  $(lyr).css({
                      transition: 'all 0.35s cubic-bezier(.34,1.56,.64,1)',
                      opacity: 1,
                      transform: 'scale(1) translateY(0)'
                  });
              }, 10);
          },
          content: '<div style="display:flex;align-items:stretch;overflow:hidden;border-radius:12px;box-shadow:0 8px 30px rgba(0,0,0,0.10),0 2px 8px rgba(0,0,0,0.06);background:' + bg + ';border:1px solid ' + (isErr ? '#ffccc7' : '#b7eb8f') + ';">' +
              /* 左側色條 */
              '<div style="width:4px;flex-shrink:0;background:linear-gradient(180deg,' + accent + ',' + (isErr ? '#ff7875' : '#73d13d') + ');"></div>' +
              /* 主體 */
              '<div style="display:flex;align-items:center;gap:12px;padding:14px 24px 14px 18px;white-space:normal;word-break:break-word;max-width:80vw;">' +
                  /* 圖標圓環 */
                  '<span style="width:30px;height:30px;border-radius:50%;background:' + accent + '1a;display:flex;align-items:center;justify-content:center;flex-shrink:0;">' +
                      '<i class="fa ' + icon + '" style="color:' + accent + ';font-size:16px;"></i>' +
                  '</span>' +
                  /* 文字 */
                  '<span style="color:' + txtCol + ';font-size:14px;font-weight:500;letter-spacing:0.01em;">' + text + '</span>' +
              '</div>' +
          '</div>'
      });
  }

  // 通用後台表單 AJAX 提交（jQuery 統一攔截，不依賴 layui form.on('submit()')）
  // 處理所有 POST 表單（包含 layui-form），用 _clicked 按鈕判斷操作類型
  $(document).on('submit', 'form:not(#dologin)', function(e) {
      var $form = $(this);
      // GET 表單（搜尋/篩選）直接原生提交
      var method = ($form.attr('method') || 'POST').toUpperCase();
      if (method === 'GET') return true;
      // 跳過有 lay-filter 的按鈕（已由 layui 專屬 handler 處理）
      var $btn = $form.find('button._clicked');
      if ($btn.length && $btn.attr('lay-filter')) return true;

      e.preventDefault();
      // 修復：layui form.getValue() 在點擊 lay-submit 時會將 name="field[]" 重命名為
      // name="field[0]"、name="field[1]" 等，導致 FormData 發送索引鍵名而非 [] 格式，
      // 後端 PostFormArray("field[]") 取不到值。在建立 FormData 前還原為 [] 格式。
      $form.find('input[name],select[name],textarea[name]').each(function(){
          var m = this.name.match(/^(.+)\[\d+\]$/);
          if (m) { this.name = m[1] + '[]'; }
      });
      // 用 FormData 保持 array 欄位名原樣（list[] 而非 list[0]）
      var formData = new FormData($form[0]);
      // 手動附加被點擊的 submit 按鈕值（FormData 不包含按鈕）
      if ($btn.length && $btn.attr('name')) {
          formData.append($btn.attr('name'), $btn.val());
      }

      $.ajax({
          type: 'POST',
          url: $form.attr('action'),
          dataType: 'json',
          data: formData,
          processData: false,
          contentType: false,
          success: function(res) {
              if (res.code == 1) {
                  showNotify(res.msg || '操作成功', 'success');
                  var returnto = $form.find('input[name="returnto"]').val();
                  if (returnto) {
                      setTimeout(function(){ window.location.href = returnto; }, 1500);
                  } else if (res.tourl && res.tourl != '') {
                      setTimeout(function(){ window.location.href = res.tourl; }, 1500);
                  }
              } else {
                  showNotify(res.msg || '操作失敗', 'error');
              }
          },
          error: function() {
              showNotify('請求發生錯誤', 'error');
          }
      });
      return false;
  });
  
  // 記錄點擊的 submit 按鈕（用於取得按鈕的 name/value）
  $(document).on('click', 'button[lay-submit], button[type=submit]', function() {
    $(this).closest('form').find('button[type=submit], button[lay-submit]').removeClass('_clicked');
    $(this).addClass('_clicked');
  });

  // ─── 通用刪除按鈕（設計：左側圖標區 + 漸進式確認） ───
  $(document).on('click', '.btn-del', function(e) {
      e.preventDefault();
      var $btn = $(this);
      var url = $btn.data('url');

      // 優先使用 data-name 屬性，否則自動提取列表行中的名稱（第二個 td 的內容）
      var targetName = $btn.data('name') || $btn.closest('tr').find('td').eq(1).text().trim() || '此項';

      layer.open({
          type: 1,
          title: false, // 自定義標題在 content 內
          area: ['440px', 'auto'],
          shadeClose: true,
          anim: 2,
          shade: [0.4, '#000'],
          btn: ['確認刪除', '取消'],
          btnAlign: 'c',
          yes: function(index) {
              $.ajax({
                  type: 'GET',
                  url: url,
                  dataType: 'json',
                  beforeSend: function() {
                      layer.load(2);
                  },
                  success: function(res) {
                      layer.closeAll('loading');
                      if (res.code == 1) {
                          showNotify(res.msg || '刪除成功', 'success');
                          setTimeout(function(){ location.reload(); }, 1200);
                      } else {
                          showNotify(res.msg || '刪除失敗', 'error');
                      }
                  },
                  error: function() {
                      layer.closeAll('loading');
                      showNotify('請求發生錯誤', 'error');
                  }
              });
              layer.close(index);
          },
          // 按鈕樣式自定義（LayUI layer 的 btn 支援 css class）
          btnClass: ['layui-btn-danger'],
          content:
              '<div style="padding:0;">' +
                  /* 頂部：大圖標 + 警示色背景 */
                  '<div style="text-align:center;padding:28px 0 18px;background:linear-gradient(135deg,#fff2f0 0%,#fff 70%);border-radius:6px 6px 0 0;">' +
                      '<span style="display:inline-flex;width:64px;height:64px;border-radius:50%;background:linear-gradient(135deg,#ff4d4f,#ff7875);box-shadow:0 6px 20px rgba(255,77,79,0.30);align-items:center;justify-content:center;">' +
                          '<i class="fa fa-trash" style="color:#fff;font-size:28px;"></i>' +
                      '</span>' +
                      '<div style="margin-top:16px;font-size:18px;font-weight:700;color:#262626;">確認刪除</div>' +
                      '<div style="margin-top:6px;font-size:14px;color:#8c8c8c;">確定要刪除 <strong style="color:#ff4d4f;">' + targetName + '</strong> 嗎？</div>' +
                  '</div>' +
                  /* 分隔線 */
                  '<div style="margin:0 24px;border-top:1px solid #f0f0f0;"></div>' +
                  /* 底部警告區域 */
                  '<div style="padding:16px 24px 22px;display:flex;align-items:flex-start;gap:10px;">' +
                      '<i class="fa fa-info-circle" style="color:#faad14;font-size:16px;flex-shrink:0;margin-top:1px;"></i>' +
                      '<span style="font-size:13px;line-height:1.6;color:#8c8c8c;">此操作將永久刪除該資料，<span style="color:#ff4d4f;font-weight:600;">不可撤銷</span>。請在操作前確認已備份重要資料。</span>' +
                  '</div>' +
              '</div>'
      });
      return false;
  });

  var sitedir=$('#sitedir').data('sitedir');
  var uploadurl = $("#preurl").data('preurl')+'/index/upload';
  
  // 單圖片上傳（整合 Squoosh 前端壓縮，大圖自動彈出對比框）
  var uploadInst = upload.render({
	elem: '.upload'
	,url: uploadurl
	,field: 'upload'
	,multiple: false
	,accept: 'images'
	,acceptMime: 'image/*'
	,auto: false
	,choose: function(obj){
		var item = this.item;  // 當前點擊的上傳按鈕 DOM
		obj.preview(function(index, file, result){
			var cfg = (typeof gbootImageConfig !== 'undefined') ? gbootImageConfig : {};
			var enable = parseInt(cfg.enable) === 1;
			var warnSize = parseInt(cfg.warnSize) || 1024;
			var fileSizeKB = file.size / 1024;

			if (enable && fileSizeKB > warnSize && typeof layui.squoosh !== 'undefined') {
				// 大圖片 → Squoosh 壓縮對比
				layui.squoosh.open(file, function(compressedFile){
					doImageUpload(compressedFile, item);
				});
			} else {
				// 小圖片或未啟用 → 直接上傳
				doImageUpload(file, item);
			}
		});
	}
  });

  // 單圖片 AJAX 上傳（壓縮後或原圖統一走此函數）
  function doImageUpload(file, item){
		var $btn = $(item);
		var des = $btn.data('des');
		var needWatermark = $btn.hasClass('watermark');
		var url = uploadurl;
		if (needWatermark) url += '/watermark/1';

		var formData = new FormData();
		formData.append('upload', file);

		layer.load();
		$.ajax({
			url: url,
			type: 'POST',
			data: formData,
			processData: false,
			contentType: false,
			success: function(res){
				layer.closeAll('loading');
				if (res.code == 1) {
					$('#' + des).val(res.data[0]);
					$('#' + des + '_box').html("<dl><dt><img src='" + sitedir + res.data[0] + "' data-url='" + res.data[0] + "' ></dt><dd>刪除</dd></dl>");
					layer.msg('上傳成功', {icon: 1});
				} else {
					layer.msg('上傳失敗：' + res.data, {icon: 2});
				}
			},
			error: function(){
				layer.closeAll('loading');
				layer.msg('上傳發生錯誤', {icon: 5});
			}
		});
  }
  
   //執行多圖片上傳實例
   var files='';
   var html='';
   var html2='';
   var uploadsInst = upload.render({
	elem: '.uploads' //綁定元素
	,url: uploadurl //上傳接口
	,field: 'upload' //欄位名稱
	,multiple: true//多檔案上傳
	,accept: 'images' //接收檔案類型 images（圖片）、file（所有檔案）、video（影片）、audio（音訊）
	,acceptMime: 'image/*'
	,before: function(obj){ 
	   //判斷是否需要加水印
       if($(this.item).hasClass('watermark')){
	  	 uploadsInst.config.url=uploadurl+'/watermark/1';//改變URL
	   }
	   layer.load(); //上傳loading
	}
	,done: function(res){
	   if(res.code==1){
		   if(files){
			   files+=','+res.data[0];
		   }else{
			   files+=res.data[0];
		   }
		   html += "<dl><dt><img src='"+sitedir+res.data[0]+"' data-url='"+res.data[0]+"'></dt><dd>刪除</dd>" +
		   		"<dt><input type='text' name='picstitle[]' style='width:95%' /></dt>"+		
		   		"</dl>";
		   html2 += "<dl><dt><img src='"+sitedir+res.data[0]+"' data-url='"+res.data[0]+"'></dt><dd>刪除</dd>" +	"</dl>";
	   }else{
		   layer.msg('有檔案上傳失敗：'+res.data, {icon: 2}); 
	   } 
	}
  	,allDone: function(obj){
  		var item = this.item;
  	    var des=$(item).data('des');
  	    
  	    layer.closeAll('loading'); //關閉loading
	    if(files!=''){
	       if($('#'+des).val()){
	    	   $('#'+des).val($('#'+des).val()+','+files); 
	       }else{
	    	   $('#'+des).val(files); 
	       }
	       if(des=='pics'){
	    	   $('#'+des+'_box').append(html); 
	       }else{
	    	   $('#'+des+'_box').append(html2); 
	       }
	 	   layer.msg('成功上傳'+obj.successful+'個檔案！', {icon: 1}); 
	 	   files='';
	 	   html='';
	 	   html2='';
	    }else{
	 	   layer.msg('全部上傳失敗！', {icon: 2}); 
	    }
	    
	 }
	,error: function(){
		layer.closeAll('loading'); //關閉loading
		layer.msg('上傳發生錯誤！', {icon: 5}); 
	}
  });
	
  //圖片頁面刪除功能
  $('.pic').on("click",'dl dd',function(){
	  var id=$(this).parents('.pic').attr('id');
	  var url=$(this).siblings('dt').find('img').data('url');
	  var input=$('#'+id.replace('_box',''));
	  var value = input.val();
	  value = value.replace(url,'');
	  value = value.replace(/^,/, '');
	  value = value.replace(/,$/, '');
	  value = value.replace(/,,/, ',');
      input.val(value);
	  $(this).parents('dl').remove();
  });
  
  //執行附件上傳實例
  var uploadFileInst = upload.render({
	elem: '.file' //綁定元素
	,url: uploadurl //上傳接口
	,field: 'upload' //欄位名稱
	,multiple: false //多檔案上傳
	,accept: 'file' //接收檔案類型 images（圖片）、file（所有檔案）、video（影片）、audio（音訊）
	,before: function(obj){ 
		layer.load(); //上傳loading
	}
	,done: function(res){
	   var item = this.item;
	   var des=$(item).data('des');
	   layer.closeAll('loading'); //關閉loading
	   if(res.code==1){
		   $('#'+des).val(res.data[0]); 
		   layer.msg('上傳成功！', {icon: 1}); 
	   }else{
		   layer.msg('上傳失敗：'+res.data, {icon: 2}); 
	   }
	}
	,error: function(){
		layer.closeAll('loading'); //關閉loading
		layer.msg('上傳發生錯誤！', {icon: 5}); 
	}
  });
  
  //使用多日期控件
  useLayDateMultiple('year','year');
  useLayDateMultiple('month','month');
  useLayDateMultiple('time','time');
  useLayDateMultiple('date','date');
  useLayDateMultiple('datetime','datetime');

  //選擇模型切換模板
   form.on('select(model)', function(data){
	  var elem = data.elem;
	  var type = $(elem).find("option:selected").data('type');
	  var listtpl = $(elem).find("option:selected").data('listtpl');
	  var contenttpl = $(elem).find("option:selected").data('contenttpl');
	  
	  $(elem).parents('form').find("#type").val(type);
	  addOptionValue("listtpl",listtpl,listtpl);
	  addOptionValue("contenttpl",contenttpl,contenttpl);
	  $(elem).parents('form').find("#listtpl").val(listtpl);
	  $(elem).parents('form').find("#contenttpl").val(contenttpl);
	  form.render(null, 'sort'); 
	}); 
   
});

//日期控件函數
function useLayDateMultiple(cls,type) {
	layui.use('laydate', function() {
		var laydate = layui.laydate;
		lay('.' + cls).each(function() {
			laydate.render({
				elem : this,
				type : type,
			});
		});
	});
} 

layui.use(['element','upload','laydate','form'], function(){
  var element = layui.element;
  var upload = layui.upload;
  var laydate = layui.laydate;
  var form = layui.form;
  
  //获取hash来切换选项卡，假设当前地址的hash为lay-id对应的值
  var layid = location.hash.replace(/^#tab=/, '');
  element.tabChange('tab', layid); //假设当前地址为：http://a.com#test1=222，那么选项卡会自动切换到“发送消息”这一项
  
  //监听Tab切换，以改变地址hash值
  element.on('tab(tab)', function(){
	var clayid=this.getAttribute('lay-id');
	if(clayid){
		location.hash = 'tab='+ clayid;
		$('.page').find('a').each(function(index,element){//避免tab翻页问题
			var url=$(this).attr('href');
			if(url.indexOf('tab=')==-1){
				$(this).attr('href', url+'#tab='+ clayid);
			}else{
				$(this).attr('href', url.replace(/tab=[\w]+/, 'tab='+ clayid));
			}
        });
	}
  });
  
  //跳转
	form.on('select(tourl)', function(data){
		window.location.href= data.value;
	}); 

  
  //提示
  $(".tips").on("mouseover",function(){
	layer.tips($(this).data('content'), this);
  }) 
 
  //用户登录验证
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
				layer.msg("登录成功！", {icon: 1});
				window.location.href = response.data;
			} else {
				form.find("#checkcode").val("");
				$('#codeimg').click();//更新验证码
				layer.msg("登录失败：" + response.data, {icon: 5});
			} 
      },
      error:function(xhr,status,error){
    	  layer.msg("登录请求发生错误!", {icon: 5});
    	  $('#note').html('登录请求失败，请检查网络连接或稍后重试。');
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

  // 通用后台表单 AJAX 提交
  // 使用 layui 的 form.on('submit()') 攔截 lay-submit 按鈕提交
  // 原因：layui 驗證通過後調用 formElem.submit()（原生方法），不觸發 jQuery delegated submit 事件
  form.on('submit()', function(data) {
      var $form = $(data.form);
      if ($form.attr('id') === 'dologin') return true; // 跳過登錄表單
      // GET 表單（搜索/篩選）直接原生提交，跳過 LayUI AJAX 處理
      var method = ($form.attr('method') || 'POST').toUpperCase();
      if (method === 'GET') {
          $form[0].submit();
          return false;
      }
      // 跳過有 lay-filter 的按鈕（已由專屬 handler 處理）
      var $btn = $(data.elem);
      if ($btn.attr('lay-filter')) return true;

      var formData = $form.serialize();

      // 確保被點擊的 submit 按鈕值被包含
      var clickedBtn = $form.find('button[lay-submit]._clicked, button[type=submit]._clicked');
      if (!clickedBtn.length) clickedBtn = $form.find('button[lay-submit], button[type=submit]').last();
      if (clickedBtn.length && clickedBtn.attr('name')) {
          formData += '&' + encodeURIComponent(clickedBtn.attr('name')) + '=' + encodeURIComponent(clickedBtn.val());
      }

      $.ajax({
          type: $form.attr('method') || 'POST',
          url: $form.attr('action'),
          dataType: 'json',
          data: formData,
          success: function(res) {
              if (res.code == 1) {
                  showNotify(res.msg || '操作成功', 'success');
                  var returnto = $form.find('input[name="returnto"]').val();
                  if (returnto) {
                      setTimeout(function(){ window.location.href = returnto; }, 1500);
                  }
              } else {
                  showNotify(res.msg || '操作失敗', 'error');
              }
          },
          error: function() {
              showNotify('请求发生错误', 'error');
          }
      });
      return false; // 阻止 layui 原生 formElem.submit()
  });

  // 排序表單（非 layui-form）的 AJAX 攔截
  $(document).on('submit', 'form:not(#dologin):has(button[value=sorting]):not(.layui-form)', function(e) {
      var $form = $(this);
      var activeEl = $(document.activeElement);
      var isSortingBtn = activeEl.is('button[value=sorting]') || $form.find('button[value=sorting]._clicked').length > 0;
      if (!isSortingBtn) return true; // 非排序按鈕（如批量刪除）正常提交

      e.preventDefault();
      var formData = $form.serialize();
      // serialize 不包含 submit 按鈕值，手動補上
      formData += '&submit=sorting';
      $.ajax({
          type: $form.attr('method') || 'POST',
          url: $form.attr('action'),
          dataType: 'json',
          data: formData,
          success: function(res) {
              if (res.code == 1) {
                  showNotify(res.msg || '操作成功', 'success');
              } else {
                  showNotify(res.msg || '操作失敗', 'error');
              }
          },
          error: function() {
              showNotify('请求发生错误', 'error');
          }
      });
      return false;
  });
  
  // 记录点击的 submit 按钮（用于获取按钮的 name/value）
  $(document).on('click', 'button[lay-submit], button[type=submit]', function() {
    $(this).closest('form').find('button[type=submit], button[lay-submit]').removeClass('_clicked');
    $(this).addClass('_clicked');
  });

  // ─── 通用刪除按鈕（設計：左側圖標區 + 漸進式確認） ───
  $(document).on('click', '.btn-del', function(e) {
      e.preventDefault();
      var $btn = $(this);
      var url = $btn.data('url');

      // 自動提取列表行中的名稱（第二個 td 的內容）
      var targetName = $btn.closest('tr').find('td').eq(1).text().trim() || '此項';

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
                      showNotify('请求发生错误', 'error');
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
                      '<span style="font-size:13px;line-height:1.6;color:#8c8c8c;">此操作將永久刪除該數據，<span style="color:#ff4d4f;font-weight:600;">不可撤銷</span>。請在操作前確認已備份重要數據。</span>' +
                  '</div>' +
              '</div>'
      });
      return false;
  });

  var sitedir=$('#sitedir').data('sitedir');
  var uploadurl = $("#preurl").data('preurl')+'/index/upload';
  
  //执行单图片实例
  var uploadInst = upload.render({
	elem: '.upload' //绑定元素
	,url: uploadurl //上传接口
	,field: 'upload' //字段名称
	,multiple: false //多文件上传
	,accept: 'images' //接收文件类型 images（图片）、file（所有文件）、video（视频）、audio（音频）
	,acceptMime: 'image/*'
    ,before: function(obj){ 
       //判断是否需要加水印
       if($(this.item).hasClass('watermark')){
	  	 uploadInst.config.url=uploadurl+'/watermark/1';//改变URL
	   }
	   layer.load(); //上传loading
	}
	,done: function(res){
	   var item = this.item;
	   var des=$(item).data('des');
	   layer.closeAll('loading'); //关闭loading
	   if(res.code==1){
		   $('#'+des).val(res.data[0]); 
		   $('#'+des+'_box').html("<dl><dt><img src='"+sitedir+res.data[0]+"' data-url='"+res.data[0]+"' ></dt><dd>删除</dd></dl>"); 
		   layer.msg('上传成功！'); 
	   }else{
		   layer.msg('上传失败：'+res.data); 
	   }
	}
	,error: function(){
		layer.closeAll('loading'); //关闭loading
		layer.msg('上传发生错误!'); 
	}
  });
  
   //执行多图片上传实例
  var files='';
  var html='';
  var html2='';
  var uploadsInst = upload.render({
	elem: '.uploads' //绑定元素
	,url: uploadurl //上传接口
	,field: 'upload' //字段名称
	,multiple: true//多文件上传
	,accept: 'images' //接收文件类型 images（图片）、file（所有文件）、video（视频）、audio（音频）
	,acceptMime: 'image/*'
	,before: function(obj){ 
	   //判断是否需要加水印
       if($(this.item).hasClass('watermark')){
	  	 uploadsInst.config.url=uploadurl+'/watermark/1';//改变URL
	   }
	   layer.load(); //上传loading
	}
	,done: function(res){
	   if(res.code==1){
		   if(files){
			   files+=','+res.data[0];
		   }else{
			   files+=res.data[0];
		   }
		   html += "<dl><dt><img src='"+sitedir+res.data[0]+"' data-url='"+res.data[0]+"'></dt><dd>删除</dd>" +
		   		"<dt><input type='text' name='picstitle[]' style='width:95%' /></dt>"+		
		   		"</dl>";
		   html2 += "<dl><dt><img src='"+sitedir+res.data[0]+"' data-url='"+res.data[0]+"'></dt><dd>删除</dd>" +	"</dl>";
	   }else{
		   layer.msg('有文件上传失败：'+res.data); 
	   } 
	}
  	,allDone: function(obj){
  		var item = this.item;
  	    var des=$(item).data('des');
  	    
  	    layer.closeAll('loading'); //关闭loading
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
	 	   layer.msg('成功上传'+obj.successful+'个文件！'); 
	 	   files='';
	 	   html='';
	 	   html2='';
	    }else{
	 	   layer.msg('全部上传失败！'); 
	    }
	    
	 }
	,error: function(){
		layer.closeAll('loading'); //关闭loading
		layer.msg('上传发生错误！'); 
	}
  });
	
  //图片页面删除功能
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
  
  //执行附件上传实例
  var uploadFileInst = upload.render({
	elem: '.file' //绑定元素
	,url: uploadurl //上传接口
	,field: 'upload' //字段名称
	,multiple: false //多文件上传
	,accept: 'file' //接收文件类型 images（图片）、file（所有文件）、video（视频）、audio（音频）
	,before: function(obj){ 
		layer.load(); //上传loading
	}
	,done: function(res){
	   var item = this.item;
	   var des=$(item).data('des');
	   layer.closeAll('loading'); //关闭loading
	   if(res.code==1){
		   $('#'+des).val(res.data[0]); 
		   layer.msg('上传成功！'); 
	   }else{
		   layer.msg('上传失败：'+res.data); 
	   }
	}
	,error: function(){
		layer.closeAll('loading'); //关闭loading
		layer.msg('上传发生错误！'); 
	}
  });
  
  //使用多日期控件
  useLayDateMultiple('year','year');
  useLayDateMultiple('month','month');
  useLayDateMultiple('time','time');
  useLayDateMultiple('date','date');
  useLayDateMultiple('datetime','datetime');

  //选择模型切换模板
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

//日期控件函数
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

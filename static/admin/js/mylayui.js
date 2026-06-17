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
  
  // 通用后台表单 AJAX 提交（排除登录表单）
  // 拦截范围：layui-form 表单 + 点击了排序按钮的任何表单
  $(document).on('submit', 'form.layui-form:not(#dologin), form:not(#dologin):has(button[value=sorting])', function(e) {
    var form = $(this);

    // ── 检测被点击的提交按钮 ──
    var activeEl = $(document.activeElement);
    var clickedBtn;
    if (activeEl.is('button[type=submit], button[lay-submit], input[type=submit]') && form[0].contains(activeEl[0])) {
        clickedBtn = activeEl;
    } else {
        clickedBtn = form.find('button[type=submit]._clicked, button[lay-submit]._clicked');
    }

    // ── 非 layui-form 表单：仅当点击排序按钮时才 AJAX，其他按钮（如批量删除）正常提交 ──
    var isLayuiForm = form.hasClass('layui-form');
    if (!isLayuiForm) {
        var isSortingBtn = (clickedBtn && clickedBtn.length && clickedBtn.val() === 'sorting')
                        || form.find('input[type=hidden][name=submit][value=sorting]').length > 0;
        if (!isSortingBtn) return true; // 正常提交（如批量删除）
    }

    e.preventDefault();
    var formData = form.serialize();

    // 确保被点击的 submit 按钮值被包含（serialize 不包含 button 值）
    if (!clickedBtn || !clickedBtn.length) clickedBtn = form.find('button[lay-submit], button[type=submit]').last();
    if (clickedBtn && clickedBtn.attr('name')) {
        formData += '&' + encodeURIComponent(clickedBtn.attr('name')) + '=' + encodeURIComponent(clickedBtn.val());
    }

    // ── 顶部居中通知 ──
    function showNotify(html, type) {
        layer.open({
            type: 1,
            title: false,
            closeBtn: 0,
            shade: 0,
            area: 'auto',
            offset: '20px',
            anim: 0,
            time: type === 'error' ? 3000 : 2000,
            content: '<div style="padding:12px 24px;border-radius:8px;background:' +
                (type === 'error' ? '#fff3f0' : '#f6ffed') +
                ';border:1px solid ' + (type === 'error' ? '#ffccc7' : '#b7eb8f') +
                ';font-size:14px;white-space:nowrap;">' + html + '</div>'
        });
    }

    $.ajax({
        type: form.attr('method') || 'POST',
        url: form.attr('action'),
        dataType: 'json',
        data: formData,
        success: function(res) {
            if (res.code == 1) {
                showNotify('<i class="fa fa-check-circle" style="color:#52c41a;margin-right:8px"></i>' + (res.msg || '操作成功'), 'success');
                var returnto = form.find('input[name="returnto"]').val();
                if (returnto) {
                    setTimeout(function(){ window.location.href = returnto; }, 1500);
                }
            } else {
                showNotify('<i class="fa fa-exclamation-circle" style="color:#ff4d4f;margin-right:8px"></i>' + (res.data || res.msg || '操作失败'), 'error');
            }
        },
        error: function() {
            showNotify('<i class="fa fa-exclamation-triangle" style="color:#ff4d4f;margin-right:8px"></i>请求发生错误', 'error');
        }
    });
    return false;
  });
  
  // 记录点击的 submit 按钮（用于获取按钮的 name/value）
  $(document).on('click', 'button[lay-submit], button[type=submit]', function() {
    $(this).closest('form').find('button[type=submit], button[lay-submit]').removeClass('_clicked');
    $(this).addClass('_clicked');
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

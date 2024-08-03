(function () {
  let query_hash = window.location.hash.replace(/^#/, "");
  let $ = mdui.$;
  const well_known_text_mime_types = [
    "text/plain",
    "text/html",
    "text/css",
    "text/javascript",
    "application/json",
    "application/xml",
    "application/xhtml+xml",
    "application/rss+xml",
    "application/atom+xml",
    "application/mathml+xml",
    "application/ecmascript",
    "application/x-javascript",
    "application/x-latex",
    "application/x-markdown",
    "application/x-yaml"
  ];
  $(function () {
    document.body.addEventListener("drop", function (e) {
      e.preventDefault();
      e.stopPropagation();
    });
    document.body.addEventListener("dragover", function (e) {
      e.preventDefault();
    });
    (function new_paste() {
      const text_input = $("#text-input");
      const file_input = $("#file-input");
      const file_paste = $("#file-paste");
      const drop_file_overlay = $(".paste-file-drop-overlay");
      const paste_password = $("#paste-password");
      const paste_expire = $("#paste-expire");
      const paste_max_access_count = $("#paste-max-access-count");
      const paste_uuid = $("#paste-uuid");
      const paste_short_url = $("#paste-short_url");
      const paste_delete_if_expired = $("#paste-delete-if-expired");
      const paste_delete = $("#paste-delete");
      const paste_update = $("#paste-update");
      const paste_submit = $("#paste-submit");
      const paste_load = $("#paste-load-from-file");

      const file_paste_icon = $("#file-paste-icon");
      const file_paste_preview = $("#file-paste-preview");
      const file_paste_filename = $("#file-paste-filename");
      const file_paste_progress = $("#file-paste-progress");
      const file_paste_progress_bar = $("#file-paste-progress-bar");
      const file_paste_progress_text = $("#file-paste-progress-text");

      const new_paste_result = $("#new-paste-result");
      const new_paste_result_link = $("#new-paste-result-link");
      const new_paste_result_raw = $("#new-paste-result-raw");
      const new_paste_result_qr_code = $("#new-paste-result-qrcode");
      const new_paste_result_copy = $("#new-paste-result-copy");

      const short_url_error = $("#short-url-error");
      let text_file = {
        filename: "",
        mime_type: ""
      };
      let paste_file = null; // file object

      let paste_preview_element = null;
      function paste_preview(file) {
        if (paste_preview_element) {
          let url = paste_preview_element.attr("src");
          if (url) {
            URL.revokeObjectURL(url);
          }
          paste_preview_element.remove();
        }
        if (file.type.startsWith("image/")) {
          paste_preview_element = $(`<img style="max-height: inherit; max-width:100%">`).attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else if (file.type.startsWith("audio/")) {
          paste_preview_element = $('<audio controls>').attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else if (file.type.startsWith("video/")) {
          paste_preview_element = $('<video controls style="max-height: inherit; max-width:100%">').attr("src", URL.createObjectURL(file)).appendTo(file_paste_preview);
        } else {
          paste_preview_element = null;
          file_paste_preview.hide();
          file_paste_icon.show();
          return
        }
        file_paste_icon.hide();
        file_paste_preview.show();
      }

      function show_file_paste_info() {
        if (!paste_file) {
          return;
        }
        file_paste_filename.text(paste_file.name + " (" + (paste_file.size / 1024 / 1024).toFixed(2).toString() + " MiB)");
        paste_preview(paste_file);
        paste_load.text("切换到文本模式").removeClass("mdui-color-theme-accent").addClass("mdui-color-blue-accent");
        text_input.parent().hide();
        file_paste.show();
      }

      function set_paste_file(file) {
        if (!file) {
          return;
        }
        // 小于 32kb 的文本文件，直接读取内容
        if (file.size <= 32 * 1024 && well_known_text_mime_types.some(type => new RegExp(type).test(file.type))) {
          const reader = new FileReader();
          reader.onload = function () {
            text_input.val(reader.result);
            text_input.get(0).dispatchEvent(new Event("input"));
          };
          reader.readAsText(file);
          paste_file = null;
          text_file.filename = file.name;
          text_file.mime_type = file.type;
          return;
        }
        paste_file = file;
        show_file_paste_info();
      }

      function switch_to_text_paste() {
        paste_file = null;
        file_input.val("");
        paste_load.text("从文件中加载").removeClass("mdui-color-blue-accent").addClass("mdui-color-theme-accent");
        file_paste.hide();
        text_input.parent().show();
      }

      paste_load.on("click", function () {
        if (!paste_file) {
          file_input.val("");
          file_input.get(0).click();
        } else {
          switch_to_text_paste();
        }
      });

      file_paste.on("click", function () {
        paste_file = null;
        file_input.val("");
        file_input.get(0).click();
      });

      file_input.on("change", function (e) {
        set_paste_file(e.target.files[0]);
      });

      text_input.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      drop_file_overlay.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      file_paste.on("dragover", function (e) {
        e.preventDefault();
        e.stopPropagation();
      });

      text_input.on("dragenter", function (e) {
        drop_file_overlay.removeClass("mdui-hidden");
      });

      function hide_drop_file_overlay(e) {
        drop_file_overlay.addClass("mdui-hidden");
      }
      drop_file_overlay.on("dragleave", hide_drop_file_overlay);

      function drop_file(e) {
        e.preventDefault();
        e.stopPropagation();
        set_paste_file(e.dataTransfer.files[0]);
        hide_drop_file_overlay(e);
      }

      text_input.on("drop", drop_file);
      drop_file_overlay.on("drop", drop_file);
      file_paste.on("drop", drop_file);

      function check_uuid(uuid) {
        return uuid.length == 0 || /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/im.test(uuid);
      }

      paste_uuid.on("input", function () {
        let uuid = paste_uuid.val();
        let uuid_valid = check_uuid(uuid);
        if (!uuid_valid) {
          paste_uuid.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
        } else {
          paste_uuid.closest(".mdui-textfield").removeClass("mdui-textfield-invalid");
        }
        if (uuid_valid && uuid.length != 0) {
          paste_delete.removeAttr("disabled");
          paste_update.removeAttr("disabled");
        } else {
          paste_delete.attr("disabled", "disabled");
          paste_update.attr("disabled", "disabled");
        }
      })

      function check_short_url(url) {
        return url.length == 0 || /^[A-Za-z0-9\\._-]+$/im.test(url);
      }

      const check_short_url_available = _.debounce(function (url) {
        $.ajax({
          method: 'GET',
          url: 'api/check_url/' + url,
        }).then(res => {
          response = JSON.parse(res);
          if (!response.available) {
            short_url_error.text("此短链接已被占用");
            paste_short_url.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
          }
        });
      }, 300);

      paste_short_url.on("input", function () {
        let url = paste_short_url.val();
        if (!check_short_url(url)) {
          short_url_error.text("仅可使用大小写字母(A-Za-z) 数字(0-9) 部分特殊符号[连字符(-) 下划线(_) 点(.)]");
          paste_short_url.closest(".mdui-textfield").addClass("mdui-textfield-invalid");
        } else {
          paste_short_url.closest(".mdui-textfield").removeClass("mdui-textfield-invalid");
        }
        check_short_url_available(url);
      })

      function check_allow_delete_if_expired() {
        if (paste_expire.val() != "0" || parseInt(paste_max_access_count.val()) > 0) {
          paste_delete_if_expired.removeAttr("disabled");
        } else {
          paste_delete_if_expired.attr("disabled", "disabled");
          paste_delete_if_expired.prop("checked", false);
        }
      }

      paste_expire.on("change", check_allow_delete_if_expired);
      paste_max_access_count.on("input", check_allow_delete_if_expired);

      paste_submit.on("click", function () {
        let text = text_input.val();
        let password = paste_password.val();
        let expire = paste_expire.val();
        let max_access_count = paste_max_access_count.val();
        let short_url = paste_short_url.val();
        let delete_if_expired = paste_delete_if_expired.prop("checked");
        let data = new FormData();
        let query_params = {};
        if (password.length != 0) {
          query_params.password = password;
        }

        if (expire != "0") {
          query_params.expire = new Date().getTime() + parseInt(expire);
        }

        if (max_access_count.length != 0) {
          if (!/^[\d]+$/m.test(max_access_count) || isNaN(parseInt(max_access_count))) {
            mdui.snackbar("最大访问次数必须为数字");
            return;
          }
          query_params.max_access_count = parseInt(max_access_count, 10);
        }

        if (short_url.length != 0) {
          if (paste_short_url.closest(".mdui-textfield").hasClass("mdui-textfield-invalid")) {
            mdui.snackbar(short_url_error.text());
            return;
          }
          query_params.short_url = short_url;
        }

        if (delete_if_expired) {
          query_params.delete_if_expired = "true";
        }

        if (paste_file) {
          data.append("c", paste_file);
        } else {
          data.append("c", new File([text], text_file.filename || "-", { type: text_file.mime_type == "" ? "text/plain" : text_file.mime_type }));
        }

        function upload_progress(e) {
          if (e.lengthComputable) {
            file_paste_progress_text.text((e.loaded / 1024 / 1024).toFixed(2) + " MiB / " + (e.total / 1024 / 1024).toFixed(2) + " MiB - " + (e.loaded / e.total * 100).toFixed(2) + "%");
            file_paste_progress_bar.css("width", (e.loaded / e.total * 100).toFixed(2) + "%");
          }
        }

        paste_submit.attr("disabled", "disabled");
        new_paste_result.css("height", "0px");
        const query_string = $.param(query_params);
        $.ajax({
          method: 'POST',
          url: '/' + query_string.length != 0 ? "?" + query_string : "",
          data: data,
          headers: {
            "Accept": "application/json"
          },
          contentType: false,
          processData: false,
          beforeSend: function (xhr) {
            xhr.upload.addEventListener("progress", upload_progress);
            file_paste_progress.css("height", "18px")
          }
        }).then(res => {
          response = JSON.parse(res);
          if (response.code != 0) {
            paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
            setTimeout(() => {
              paste_submit.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
            }, 1000);
          } else {
            paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-green-accent");
            setTimeout(() => {
              paste_submit.removeClass("mdui-color-green-accent").addClass("mdui-color-theme-accent");
            }, 1000);
          }
          raw_html = ""
          for (let [k, v] of Object.entries(response)) {
            if (k == "code" || k == "url") {
              continue;
            }
            if (raw_html.length != 0) {
              raw_html += "\n";
            }
            raw_html += `<p><strong>${k}:</strong> ${v}</p>`;
          }
          new_paste_result_raw.html(raw_html);
          new_paste_result_link.text(response.url);
          new_paste_result_link.attr("href", response.url);
          paste_uuid.val(response.uuid);
          paste_uuid.get(0).dispatchEvent(new Event("input"));
          QRCode.toCanvas(new_paste_result_qr_code.get(0), response.url, { margin: 0, width: 168 }, function () { });
          new_paste_result.css("height", new_paste_result.children().height() + "px");
        }).catch(err => {
          paste_submit.removeClass("mdui-color-theme-accent").addClass("mdui-color-red-accent");
          setTimeout(() => {
            paste_submit.removeClass("mdui-color-red-accent").addClass("mdui-color-theme-accent");
          }, 1000);
          mdui.snackbar("创建失败: " + err);
        }).finally(() => {
          paste_submit.removeAttr("disabled");
          file_paste_progress.css("height", "0px")
        });
      });

      new_paste_result_copy.on("click", function () {
        function selectAndHint() {
          let selection = window.getSelection();
          let range = document.createRange();
          range.selectNodeContents(new_paste_result_link.get(0));
          selection.removeAllRanges();
          selection.addRange(range);
          mdui.snackbar("请按 Ctrl+C 复制");
        }
        if (navigator.clipboard) {
          navigator.clipboard.writeText(new_paste_result_link.text()).then(() => {
            mdui.snackbar("已复制到剪贴板");
          }).catch(err => {
            selectAndHint();
          })
        } else {
          selectAndHint();
        }
      });
    })();
  });
})();
